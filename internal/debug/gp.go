package debug

// TOP GP 的登录与启动段。
//
// 只覆盖两件事：怎么进 shell、怎么把调试器拉起来。之后的一切 —— 命令发送、
// 提示符识别、停站判定、断点恢复、源码定位、变量解析 —— 与 T100 共用同一套
// 代码（它们认的都是 fgldb 的文本输出）。
//
// 两条与 T100 的硬差别（都是真机试出来的）：
//  1. 登录菜单是实例菜单（Please Select (1)topprod (2)toptest …），要按实例名选；
//     实例名从 $TOP/$CUST 路径推断，不能拿环境显示名（22主机 ≠ toptest）；
//  2. r.d2 要的是**去掉模块前缀**的程序名：csf_csfi300_icd → csfi300_icd。

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	gpsvc "tt/internal/gp"
)

// gpReMenu 登录器问实例的那一行。
var gpReMenu = regexp.MustCompile(`Please Select`)

// gpSafeSrcName 只允许普通模块/程序名，避免拼进 shell 的检查命令时被注入。
var gpSafeSrcName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// gpMenuOpt 匹配实例菜单里的编号与名字。
var gpMenuOpt = regexp.MustCompile(`\((\d+)\)\s*([A-Za-z0-9_]+)`)

// gpInstance 推断 GP 实例名：
//   - 优先从 $TOP / $CUST 路径里找以 top 开头的段（/u1/toptest/tiptop → toptest）；
//   - 路径里没有时才退回环境名（兼容旧配置 TOPTEST）。
//
// 环境显示名可以被用户改（TOPTEST → 22主机），实例名不能跟着改；否则登录菜单里
// 找不到 (2)toptest，登录会直接掉回 exit。
func gpInstance(env string, roots ...string) string {
	for _, root := range roots {
		for _, p := range strings.Split(strings.Trim(root, "/"), "/") {
			if strings.HasPrefix(strings.ToLower(p), "top") {
				return strings.ToLower(p)
			}
		}
	}
	return strings.ToLower(strings.TrimSpace(env))
}

// pLinkModule 从本地 GP 字典的 t_prog 查程序实际模块（p_link 回填）。
// 本地库没有/未同步时返回 false，调用方继续用表单模块，不额外连 Oracle。
func (s *Session) pLinkModule(prog string) (string, bool) {
	for i := range s.cfg.SSHs {
		if s.cfg.SSHs[i].Name == s.envName {
			return gpsvc.LookupModule(s.cfg.SSHs[i], prog)
		}
	}
	return "", false
}

// bootGP 登录 GP 服务器：等实例菜单 → 选实例 → 等 shell 提示符。
func (s *Session) bootGP(ctx context.Context) error {
	s.emitEvent(Event{Type: "log", Text: "GP: 等实例菜单（Please Select）"})
	// 自己收行而不是用 waitRegexp：那样只能知道"匹配到了"，拿不到菜单原文，
	// 也没法确认读到的是这一轮的菜单而不是缓冲区里的旧文本。
	menu, lines := s.gpReadUntil(gpReMenu, 30*time.Second)
	if !gpReMenu.MatchString(menu) {
		s.emitEvent(Event{Type: "log", Text: "GP: 实例菜单没等到；最近读到：" + tailOf(strings.Join(lines, "\n"), 300)})
		return fmt.Errorf("等实例菜单超时；最近输出：%s", tailOf(strings.Join(lines, "\n"), 300))
	}
	inst := gpInstance(s.envName, s.cfg.Workspace, s.cfg.CustRoot)
	num := gpPickInstance(menu, inst)
	s.emitEvent(Event{Type: "log", Text: "GP: 菜单原文 → " + oneLine(menu, 200)})
	if num == "" {
		s.emitEvent(Event{Type: "log", Text: "GP: 菜单里没找到实例 " + inst + "，回车走默认"})
		num = ""
	}
	s.pty.Write(num + "\r")
	time.Sleep(4 * time.Second)
	s.markShellReady()
	s.mu.Lock()
	s.booted = true
	s.mu.Unlock()
	s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("已登录 %s（TOP GP 环境 %s，实例 %s）", s.cfg.SSH.Host, s.envName, inst)})
	return nil
}

// gpReadUntil 从会话行流里读到匹配为止，返回累计文本与原始行。
func (s *Session) gpReadUntil(re *regexp.Regexp, timeout time.Duration) (string, []string) {
	deadline := time.After(timeout)
	var b strings.Builder
	var lines []string
	for {
		select {
		case ln := <-s.lines:
			lines = append(lines, ln)
			b.WriteString(ln)
			b.WriteString("\n")
			if re.MatchString(b.String()) {
				return b.String(), lines
			}
		case <-deadline:
			return b.String(), lines
		}
	}
}

// oneLine 压成一行（日志里看菜单原文用）。
func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

// tailOf 取尾部若干字符。
func tailOf(s string, n int) string {
	if len(s) <= n {
		return oneLine(s, n)
	}
	return oneLine(s[len(s)-n:], n)
}

// gpPickInstance 从菜单文本里找实例对应的编号。
func gpPickInstance(text, inst string) string {
	for _, m := range gpMenuOpt.FindAllStringSubmatch(text, -1) {
		if strings.EqualFold(m[2], inst) {
			return m[1]
		}
	}
	return ""
}

// gpModuleOf 从作业编号取模块：取第一个下划线之前那段。
//
// 模块**不一定是三位**（csf / asf / aap / gzz …），所以不能用 prog[:3] 硬切。
// 没有下划线时才退回前三位。
func gpModuleOf(prog string) string {
	if i := strings.Index(prog, "_"); i > 0 {
		return strings.ToLower(prog[:i])
	}
	if len(prog) > 3 {
		return strings.ToLower(prog[:3])
	}
	return strings.ToLower(prog)
}

// gpShortProg 去掉模块前缀：csf_csfi300_icd → csfi300_icd。
func gpShortProg(prog string) string {
	if len(prog) > 4 && prog[3] == '_' {
		return prog[4:]
	}
	return prog
}

// startRunGP 进模块的 4gl 目录并起调试器，等到 (fgldb)。
func (s *Session) startRunGP(ctx context.Context, launchProg string) error {
	log.Printf("[debug] 会话 %s 进 startRunGP prog=%q", s.ID, launchProg)
	// 优先用启动请求里显式给的模块（与 T100 的 resolveJobWith 一致）；
	// 只有未给模块时才按程序名推导。
	mod := strings.ToLower(strings.TrimSpace(s.Module))
	if mod == "" {
		mod = gpModuleOf(launchProg)
	}
	// 表单模块只是提示；p_link 才是权威。本地 t_prog 命中且与表单不同时，
	// 用实际模块启动，并在会话时间线里明确提醒（否则用户会以为填的模块生效了）。
	if actual, ok := s.pLinkModule(launchProg); ok && !strings.EqualFold(actual, mod) {
		s.emitEvent(Event{Type: "log", Text: fmt.Sprintf(
			"GP: p_link 显示 %s 实际属于模块 %s，不是表单填的 %s；已按 %s 启动",
			launchProg, actual, mod, actual)})
		mod = actual
		s.Module = actual
	}
	// 用 $TOP/$CUST 这对登录环境变量：r.d2 内部也按它们找 42r/42f，
	// 换成绝对路径它就解析不到程序了（实测过一次）。
	setup := fmt.Sprintf("cd $TOP/%s/4gl || cd $CUST/%s/4gl\r\n", mod, mod)
	s.emitEvent(Event{Type: "log", Text: "GP: 进模块目录 " + mod})
	s.pty.Write(setup)
	time.Sleep(2 * time.Second)

	// 先确认模块目录里真有这个程序。否则 r.d2 只会在 shell 里报错，
	// 而这里会一直等 (fgldb) 90 秒；p_link 选错模块时用户看到的就是“卡住”。
	short := gpShortProg(launchProg)
	if !gpSafeSrcName.MatchString(mod) || !gpSafeSrcName.MatchString(short) {
		return fmt.Errorf("GP 模块/程序名含有不安全字符：%q/%q", mod, short)
	}
	checkCmd := fmt.Sprintf(
		`if [ -f "$TOP/%s/4gl/%s.4gl" ] || [ -f "$CUST/%s/4gl/%s.4gl" ]; then printf 'GP_SRC_%%s\n' OK; else printf 'GP_SRC_%%s\n' MISSING; fi`+"\r\n",
		mod, short, mod, short)
	s.pty.Write(checkCmd)
	checkOut, _ := s.gpReadUntil(regexp.MustCompile(`GP_SRC_(OK|MISSING)`), 5*time.Second)
	if strings.Contains(checkOut, "GP_SRC_MISSING") || !strings.Contains(checkOut, "GP_SRC_OK") {
		return fmt.Errorf("模块 %s 下找不到 %s.4gl（p_link 解析结果可能不对，请检查 p_link / 模块）", mod, short)
	}
	s.emitEvent(Event{Type: "log", Text: "GP: 源码存在 " + mod + "/" + short + ".4gl"})

	runCmd := "r.d2 " + short
	log.Printf("[debug] GP 启动命令 = %q（原始作业 %q，模块 %q）", runCmd, launchProg, mod)
	log.Printf("[debug] 会话 %s TOP GP 启动: %s", s.ID, runCmd)
	s.mu.Lock()
	s.shellReady = false
	s.mu.Unlock()
	s.pty.Write(runCmd + "\r")
	log.Printf("[debug] 会话 %s 已发 r.d2，等 (fgldb)", s.ID)
	if err := s.waitBarePrompt(90*time.Second, "(fgldb) 提示符"); err != nil {
		s.emitEvent(Event{Type: "log", Text: "GP: 调试器没起来 —— " + err.Error()})
		return err
	}
	s.emitEvent(Event{Type: "log", Text: "GP: 调试器已就绪"})
	s.mu.Lock()
	s.state = StateStopped
	s.startAt = time.Now()
	s.restoring = true
	s.mu.Unlock()
	s.setStop(s.entryStopInfo())
	// 与 T100 的 startRun 收尾保持一致：设 print 上限、恢复断点，再宣告停站。
	// 少了这段，restoring 一直是 true，对外永远报 loading（实测就是这个症状）。
	if _, err := s.exec("other", fmt.Sprintf("set print elements %d", s.cfg.PrintElements), waitPrompt, 10*time.Second); err != nil {
		s.emitEvent(Event{Type: "log", Text: "set print elements 失败(不影响使用): " + err.Error()})
	}
	s.restoreBreakpoints()
	s.mu.Lock()
	s.restoring = false
	s.mu.Unlock()
	s.emitEvent(Event{Type: "state", State: string(StateStopped)})
	s.emitEvent(Event{Type: "log", Text: "TOP GP 调试会话就绪: " + s.Prog})
	return nil
}
