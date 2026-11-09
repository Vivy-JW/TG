// Package sshtun 提供 SSH 端口转发隧道:本地监听一个 TCP 端口,把连接经 SSH
// 转发到远端 host:port。用于"客户端不可达 DB、但 DB 对 SSH 服务器(或同网)可达"时,
// 客户端驱动仍以直连方式(连 127.0.0.1:本地端口)查询远程数据库。
package sshtun

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Tunnel 一个运行中的 SSH 端口转发隧道。
type Tunnel struct {
	sshCli    *ssh.Client
	listener  net.Listener
	LocalPort int
	once      sync.Once
	done      chan struct{}
	closeErr  error
}

// Options SSH 连接与转发目标参数。
type Options struct {
	Host       string
	Port       int // 0 → 22
	User       string
	Password   string
	RemoteHost string // DB 在 SSH 侧的真实地址
	RemotePort int    // DB 在 SSH 侧的真实端口
	LocalPort  int    // 0 → 自动选空闲端口
}

// Dial 建立 SSH 连接并启动本地转发;LocalPort>0 时尝试绑定该端口。
func Dial(o Options) (*Tunnel, error) {
	if o.Port == 0 {
		o.Port = 22
	}
	sshCfg := &ssh.ClientConfig{
		User:            o.User,
		Auth:            []ssh.AuthMethod{ssh.Password(o.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 与 debug 手工登录一致(内网工具)
		Timeout:         20 * time.Second,
	}
	cli, err := ssh.Dial("tcp", net.JoinHostPort(o.Host, itoa(o.Port)), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败(%s): %w", o.Host, err)
	}
	// 本地监听:指定端口或自动
	bind := "127.0.0.1:0"
	if o.LocalPort > 0 {
		bind = net.JoinHostPort("127.0.0.1", itoa(o.LocalPort))
	}
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("本地监听失败: %w", err)
	}
	t := &Tunnel{
		sshCli:    cli,
		listener:  ln,
		LocalPort: ln.Addr().(*net.TCPAddr).Port,
		done:      make(chan struct{}),
	}
	remote := net.JoinHostPort(o.RemoteHost, itoa(o.RemotePort))
	go t.acceptLoop(remote)
	return t, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func (t *Tunnel) acceptLoop(remote string) {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done: // 主动关闭
				return
			default:
			}
			t.closeErr = err
			return
		}
		go t.forward(conn, remote)
	}
}

func (t *Tunnel) forward(local net.Conn, remote string) {
	defer local.Close()
	rconn, err := t.sshCli.Dial("tcp", remote)
	if err != nil {
		return
	}
	defer rconn.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(rconn, local) }()
	go func() { defer wg.Done(); io.Copy(local, rconn) }()
	wg.Wait()
}

// Close 关闭本地监听、SSH 连接与转发。
func (t *Tunnel) Close() error {
	t.once.Do(func() {
		close(t.done)
		t.listener.Close()
		t.sshCli.Close()
	})
	return t.closeErr
}
