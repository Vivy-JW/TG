// Package live 提供"远程 ERP 库直查"的数据访问层。
// *Live 实现 db.Source:与本地 SQLite 镜像(db 包)同语义 —— 同一套字典表
// (dzea_t 等 24 张)与 JOIN SQL,直接连远程库执行,rt/rv/desc/scc/rq 的数据源
// 切到某环境时即经本包(金仓 + Oracle 双方言,见 source.go)。
//
// 设计要点:
//   - erpdb.Connector 走简单协议(simple protocol),不支持绑定参数 → 所有值经
//     QuoteLit/ValidIdent 白名单后内联,仍保持单条只读 SELECT;
//   - viaSsh 隧道:客户端不可达 DB 时先建 SSH 端口转发,连接 host 换成 127.0.0.1:本地端口。
package live

import (
	"context"

	"tt/internal/dbconfig"
	"tt/internal/erpdb"
	"tt/internal/sshtun"
)

// Live 一个远程数据源:底层连接器 + 可选 SSH 隧道(Close 一并释放)。
type Live struct {
	conn erpdb.Connector
	tun  *sshtun.Tunnel
}

// Dialect 返回底层连接方言("kingbase"|"oracle")。
func (l *Live) Dialect() string { return l.conn.Type() }

// Connector 暴露底层连接器(SelectAllSQL 等通用能力)。
func (l *Live) Connector() erpdb.Connector { return l.conn }

// Close 释放连接;若开了 SSH 隧道一并关闭。
func (l *Live) Close() {
	if l.conn != nil {
		l.conn.Close()
	}
	if l.tun != nil {
		l.tun.Close()
	}
}

// Open 按连接配置建立远程数据源。若配置了 viaSsh,先起 SSH 隧道再连本地转发端口。
func Open(ctx context.Context, c dbconfig.Connection) (*Live, error) {
	effective := c
	if c.ViaSSH != nil {
		remoteHost, remotePort := c.ViaSSH.EffectiveRemote(c.Host, c.Port)
		t, err := sshtun.Dial(sshtun.Options{
			Host:       c.ViaSSH.Host,
			Port:       c.ViaSSH.Port,
			User:       c.ViaSSH.User,
			Password:   c.ViaSSH.Password,
			RemoteHost: remoteHost,
			RemotePort: remotePort,
			LocalPort:  c.ViaSSH.BindPort,
		})
		if err != nil {
			return nil, err
		}
		// 连接器连本地转发端口(host 固定 127.0.0.1;oracle 保持 service 不变)
		effective.Host = "127.0.0.1"
		effective.Port = t.LocalPort
		conn, err := erpdb.Open(ctx, effective)
		if err != nil {
			t.Close()
			return nil, err
		}
		return &Live{conn: conn, tun: t}, nil
	}
	conn, err := erpdb.Open(ctx, effective)
	if err != nil {
		return nil, err
	}
	return &Live{conn: conn}, nil
}
