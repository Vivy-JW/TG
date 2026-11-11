// Package gpsession 管 TOP GP 的调试会话：起 PTY、进程序目录、拉 r.d2、收发 fgldb 命令。
//
// 走 PTY，原因只有一个：`r.d2` 不是 PATH 里的程序，而是**交互式 ksh 里的别名**
// （实测 `command -v r.d2` 为空，`alias` 里能看到它定义的一部分）。
// 非交互的 `ssh exec` 拿不到它，而手动 source DVM 脚本后直接 `fglrun -d` 会报 -1338
// （模块链接是 r.d2 自己做的，profile 里没有）。
//
// PTY 的坑是 ksh 的行编辑：长命令写快了会吞字符、重绘时满屏退格。解法是登录后
// 每条命令写入前先等提示符空闲（见 waitPrompt）。
//
// 与 T100 的 internal/debug 不是一回事：那边是 gendbg 语义的调试器模型（RunProg/
// LaunchRef/断点状态机），GP 这边只有 pty + r.d2，没有断点协议。所以单独一个包，
// 不塞进 internal/debug，也不占用 internal/session（那里是 T100 的序号/世代号规则）。
package gpsession

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"tt/internal/config"
	"tt/internal/host"
)

// fgldbPrompt 是 fgldb 的提示符。所有命令都以它为界。
const fgldbPrompt = "(fgldb) "

var (
	ansiRe   = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")
	lineAtRe = regexp.MustCompile(`(?m)^Line (\d+) of "?([^"\n]+)"?`)
	menuOpt  = regexp.MustCompile(`\((\d+)\)\s*([A-Za-z0-9_]+)`)
	// GP 的交互提示符形如 <toptest:/u1/toptest/toptest>；出现在缓冲末尾才算空闲
	promptTail = regexp.MustCompile(`<[^>\n]*>\s*$`)
)

// Session 一条 GP 调试会话。
type Session struct {
	ID      string `json:"id"`
	Env     string `json:"env"`
	Prog    string `json:"prog"`
	State   string `json:"state"` // starting | ready | error | closed
	Err     string `json:"error,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Created string `json:"created"`
	Dir     string `json:"dir,omitempty"`

	conn *host.SSHConn
	pty  *host.PTYSession

	mu  sync.Mutex
	buf strings.Builder
}

// Manager 当前活着的会话（按会话 ID 索引）。
//
// 一条会话是一个 telnet 级的长连接，进程退出就没了 —— 所以只放内存，不落盘。
type Manager struct {
	mu sync.Mutex
	m  map[string]*Session
}

// NewManager 建一个空的管理器。
func NewManager() *Manager { return &Manager{m: map[string]*Session{}} }

func randID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "gp" + hex.EncodeToString(b)
}

func stripAnsi(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\b", "")
	return s
}

func (s *Session) pump() {
	b := make([]byte, 4096)
	for {
		n, err := s.pty.Read(b)
		if n > 0 {
			s.mu.Lock()
			s.buf.Write(b[:n])
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) waitFor(marker string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		out := s.buf.String()
		s.mu.Unlock()
		if strings.Contains(out, marker) {
			return out, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.mu.Lock()
	tail := s.buf.String()
	s.mu.Unlock()
	return tail, fmt.Errorf("等 %q 超时，最后输出：%s", marker, stripAnsi(tail))
}

func (s *Session) take() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.buf.String()
	s.buf.Reset()
	return out
}

// waitPrompt 等到 shell 回到提示符空闲态。
//
// 每条命令写入前都必须先等它 —— 上一条命令的输出还在刷的时候写进去，
// 开头的字符会被挤掉（实测丢过 13 个字符，`r.d2 csf_csfi300_icd` 变成 `sfi300 _icd`）。
func (s *Session) waitPrompt(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		tail := stripAnsi(s.buf.String())
		s.mu.Unlock()
		if promptTail.MatchString(tail) {
			return true
		}
		time.Sleep(120 * time.Millisecond)
	}
	return false
}

// instanceOf 从工作区路径推断实例名：/u1/toptest/tiptop → toptest。
func instanceOf(env config.NamedSsh) string {
	parts := strings.Split(strings.Trim(env.Workspace, "/"), "/")
	for _, p := range parts {
		if strings.HasPrefix(strings.ToLower(p), "top") {
			return strings.ToLower(p)
		}
	}
	if len(parts) > 1 {
		return strings.ToLower(parts[1])
	}
	return ""
}

// pickInstance 从登录菜单里找实例对应的编号。
func pickInstance(text, inst string) string {
	for _, m := range menuOpt.FindAllStringSubmatch(text, -1) {
		if strings.EqualFold(m[2], inst) {
			return m[1]
		}
	}
	return ""
}

// dirCandidates 程序可能在 $CUST 或 $TOP 的 4gl / 42m 下，挨个试。
//
// 目前建会话走的是 shell 变量（见 New 的注释），这个函数留给"要报告的候选目录"用。
func dirCandidates(env config.NamedSsh, prog string) []string {
	mod := prog
	if len(mod) > 3 {
		mod = mod[:3]
	}
	var out []string
	// 顺序有讲究：先 4gl（r.d2 从源码目录起才对得上程序），42m 只作退路
	if env.CustRoot != "" {
		out = append(out, env.CustRoot+"/"+mod+"/4gl", env.CustRoot+"/"+mod+"/42m")
	}
	if env.Workspace != "" {
		out = append(out, env.Workspace+"/"+mod+"/4gl", env.Workspace+"/"+mod+"/42m")
	}
	return out
}

// New 建一条 GP 调试会话：登录 → 选实例 → 进目录 → r.d2。失败时返回的 sess 可能是半成品
// （状态 error + 错误文字），由调用方决定要不要留着重试。
func (m *Manager) New(env config.NamedSsh, prog string) (*Session, error) {
	if prog == "" {
		return nil, fmt.Errorf("TOP GP 会话必须给程序编号")
	}
	conn, err := host.Dial(config.SSHConfig{Host: env.Host, Port: env.Port, User: env.User, Password: env.Password})
	if err != nil {
		return nil, fmt.Errorf("SSH 连不上：%w", err)
	}
	pty, err := conn.NewPTY(220, 60)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("开不了终端：%w", err)
	}
	sess := &Session{
		ID: randID(), Env: env.Name, Prog: prog, State: "starting",
		Created: time.Now().Format("15:04:05"), conn: conn, pty: pty,
	}
	go sess.pump()

	// 1) 登录器先问实例
	if out, err := sess.waitFor("Please Select", 25*time.Second); err == nil {
		if num := pickInstance(out, instanceOf(env)); num != "" {
			_ = pty.Write(num + "\n")
		} else {
			_ = pty.Write("\n")
		}
	}
	sess.take()
	// 实例选完要等环境铺完、提示符出来，别按秒猜
	sess.waitPrompt(40 * time.Second)
	sess.take()

	// 2) 不做 stty：r.d2 这个包装脚本自己会读终端，把 icanon 关掉反而让它出错
	// 3) 进程序目录：**用 $TOP/$CUST 这对 shell 变量**，不要自拼绝对路径 ——
	//    r.d2 内部也按它们找 42r/42f/42m，换成绝对路径它就解析不到程序了（实测）。
	mod := prog
	if len(mod) > 3 {
		mod = mod[:3]
	}
	_ = pty.Write("cd $TOP/" + mod + "/4gl || cd $CUST/" + mod + "/4gl\n")
	sess.waitPrompt(15 * time.Second)
	sess.take()

	// 4) 起调试器：写之前再确认一次 shell 空闲
	sess.waitPrompt(10 * time.Second)
	// r.d2 要的是**去掉模块前缀**的程序名：csf_csfi300_icd → csfi300_icd
	// （用户实测：在对应 4gl 目录下敲 `r.d2+ csfi300_icd` 才起得来）
	rprog := prog
	if len(rprog) > 4 && rprog[3] == '_' {
		rprog = rprog[4:]
	}
	_ = pty.Write("r.d2 " + rprog + "\n")
	if _, err := sess.waitFor(fgldbPrompt, 60*time.Second); err != nil {
		sess.State = "error"
		sess.Err = "调试器没起来（60 秒内没看到 (fgldb) 提示符）"
		return sess, err
	}
	sess.take()
	sess.State = "ready"

	// 5) 让调试器报一次入口位置，顺便拿到源码文件名与行号
	if o, err := sess.Exec("info line main", 15*time.Second); err == nil {
		if mm := lineAtRe.FindStringSubmatch(o); mm != nil {
			sess.File = mm[2]
			fmt.Sscanf(mm[1], "%d", &sess.Line)
		}
	}
	return sess, nil
}

// Add 把会话登记进管理器（New 成功或半成功时由调用方调）。
func (m *Manager) Add(s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[s.ID] = s
}

// ByID 按 ID 找会话；没有返回 nil。
func (m *Manager) ByID(id string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.m[id]
}

// List 当前所有会话。
func (m *Manager) List() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, 0, len(m.m))
	for _, v := range m.m {
		out = append(out, v)
	}
	return out
}

// Close 收掉一条会话并从表里摘掉；ID 不存在时什么都不做。
func (m *Manager) Close(id string) bool {
	m.mu.Lock()
	s := m.m[id]
	delete(m.m, id)
	m.mu.Unlock()
	if s == nil {
		return false
	}
	s.Close()
	return true
}

// Exec 发一条 fgldb 命令，返回到下一个提示符之间的输出。
func (s *Session) Exec(cmd string, timeout time.Duration) (string, error) {
	s.take()
	if err := s.pty.Write(cmd + "\n"); err != nil {
		return "", err
	}
	raw, err := s.waitFor(fgldbPrompt, timeout)
	if err != nil {
		return "", err
	}
	out := stripAnsi(raw)
	out = strings.ReplaceAll(out, fgldbPrompt, "")
	out = strings.TrimSpace(out)
	if m := lineAtRe.FindStringSubmatch(out); m != nil {
		s.File = m[2]
		fmt.Sscanf(m[1], "%d", &s.Line)
	}
	s.take()
	return out, nil
}

// Close 收掉会话。
func (s *Session) Close() {
	if s.pty != nil {
		_ = s.pty.Write("quit\n")
		time.Sleep(200 * time.Millisecond)
		_ = s.pty.Write("exit\n")
		time.Sleep(100 * time.Millisecond)
		s.pty.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
	s.State = "closed"
}
