package debug

// 这组测试盯的是"这一轮画面去哪"(FGLSERVER)这条线。
//
// 事故背景(实测):`debug.fglserver` 是**全局**配置,而它同时被调试线(要指 GDC)和
// 无头前端实验(要指我们自己)当接线开关用。做实验时把它改成 `<本机IP>:1300`
// (端口 1300 → 前端 7700),之后每一次调试:程序照常启动、照常停站,放行之后
// GDC 里什么都不弹,而启动输出一个字都没提画面去了哪 —— 排查方向全在"GDC 是不是坏了"。
//
// 所以这里钉住两件事:
//  1. --fglserver 是**单轮**覆盖,且"空串"有语义(本轮交回 T100 自动设置),
//     不能按值判空,否则最好用的那个动作恰恰做不到;
//  2. 启动输出必须明说画面去哪,别让人靠翻配置猜。

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
)

// newWireCmd 造一个只挂 --fglserver 的命令:覆盖语义靠 Flags().Changed 判定,
// 所以必须拿真命令(在全局命令上改标志会污染别的用例)。
func newWireCmd(t *testing.T) *cobra.Command {
	t.Helper()
	old := dbgFGLServer
	t.Cleanup(func() { dbgFGLServer = old })
	c := &cobra.Command{Use: "wire"}
	c.Flags().StringVar(&dbgFGLServer, "fglserver", "", "")
	return c
}

func TestDBGWireOverridesFGLServer(t *testing.T) {
	oldZone, oldSSH, oldEnv, oldGlobal := dbgZone, dbgSSHName, dbgEnvFlag, common.Env
	t.Cleanup(func() { dbgZone, dbgSSHName, dbgEnvFlag, common.Env = oldZone, oldSSH, oldEnv, oldGlobal })
	dbgZone, dbgSSHName, dbgEnvFlag, common.Env = "", "", "", ""

	t.Run("不给开关:请求体里不该出现这个键(用配置值)", func(t *testing.T) {
		cmd := newWireCmd(t)
		body := map[string]any{}
		target := dbgWireOverrides(cmd, body)
		if _, ok := body["fglserver"]; ok {
			t.Fatalf("没给 --fglserver 却填了请求体: %v", body)
		}
		if target != "" {
			t.Fatalf("没点名环境时 target 应为空, got %q", target)
		}
	})

	t.Run("给了值:按值下发", func(t *testing.T) {
		cmd := newWireCmd(t)
		if err := cmd.Flags().Set("fglserver", "192.168.14.193:1000"); err != nil {
			t.Fatal(err)
		}
		body := map[string]any{}
		dbgWireOverrides(cmd, body)
		if got := body["fglserver"]; got != "192.168.14.193:1000" {
			t.Fatalf("覆盖值应原样下发, got %v", got)
		}
	})

	t.Run("给了空串:这是「本轮交回 T100 自动设置」,必须下发", func(t *testing.T) {
		cmd := newWireCmd(t)
		if err := cmd.Flags().Set("fglserver", ""); err != nil {
			t.Fatal(err)
		}
		body := map[string]any{}
		dbgWireOverrides(cmd, body)
		v, ok := body["fglserver"]
		if !ok {
			t.Fatal("空串必须下发(否则配置里钉死的值就没法在某一轮收回来)")
		}
		if v != "" {
			t.Fatalf("空串应原样下发, got %v", v)
		}
	})

	t.Run("环境/区域/覆盖三件事各归各的", func(t *testing.T) {
		cmd := newWireCmd(t)
		dbgZone, dbgSSHName = "36", "主机正式区"
		if err := cmd.Flags().Set("fglserver", "本机:1300"); err != nil {
			t.Fatal(err)
		}
		body := map[string]any{}
		if target := dbgWireOverrides(cmd, body); target != "主机正式区" {
			t.Fatalf("--ssh/--env 仍是环境的权威来源, got %q", target)
		}
		if body["zone"] != "36" || body["ssh"] != "主机正式区" || body["fglserver"] != "本机:1300" {
			t.Fatalf("三件事应同时下发: %v", body)
		}
	})
}

func TestDBGScreenLine(t *testing.T) {
	if got := dbgScreenLine(nil); got != "" {
		t.Fatalf("没有快照时不该打半行, got %q", got)
	}
	// 服务端给了人话版本:原样用(措辞只有一处,别在这儿再编一套)
	srv := "画面去向: FGLSERVER=10.1.25.33:1300(本轮覆盖) —— ..."
	if got := dbgScreenLine(&dbgSnapView{ScreenTarget: srv}); got != srv {
		t.Fatalf("应原样用服务端那句, got %q", got)
	}
	// 旧服务端(没有 screenTarget 字段):至少把裸值说出来,不能什么都不打
	got := dbgScreenLine(&dbgSnapView{FGLServer: "10.1.25.33:1300"})
	if !strings.Contains(got, "10.1.25.33:1300") || !strings.Contains(got, "画面去向") {
		t.Fatalf("裸值也要能说明画面去向, got %q", got)
	}
	// 自动设置(两个字段都空)时没有可说的:由服务端那句负责,这里不编
	if got := dbgScreenLine(&dbgSnapView{}); got != "" {
		t.Fatalf("没有覆盖值时不该编一行, got %q", got)
	}
}

// 点了 --fglserver 就必须认账:服务端不认这个字段时是**静默忽略**的,
// 而"以为自己改了接线、其实没改"正是这次事故的形状,所以必须判失败。
func TestDBGScreenVerify(t *testing.T) {
	snap := &dbgSnapView{FGLServer: "192.168.14.193:1000"}
	if err := dbgScreenVerify("1.2.3.4:1300", false, snap); err != nil {
		t.Fatalf("没点这个开关时不该管快照里是什么: %v", err)
	}
	if err := dbgScreenVerify("192.168.14.193:1000", true, snap); err != nil {
		t.Fatalf("生效了就不该报错: %v", err)
	}
	if err := dbgScreenVerify(" 192.168.14.193:1000 ", true, snap); err != nil {
		t.Fatalf("两侧空白不该算不一致: %v", err)
	}
	if err := dbgScreenVerify("1.2.3.4:1300", true, snap); err == nil {
		t.Fatal("点了名却没生效必须判失败(否则调用方以为画面换地方了)")
	}
	// 空串是"本轮交回 T100 自动设置":生效时快照里的值也应是空
	if err := dbgScreenVerify("", true, &dbgSnapView{}); err != nil {
		t.Fatalf("空串覆盖生效(快照里也是空)时不该报错: %v", err)
	}
	if err := dbgScreenVerify("", true, snap); err == nil {
		t.Fatal("要求交回自动设置、实际还钉着值时必须判失败")
	}
	if err := dbgScreenVerify("1.2.3.4:1300", true, nil); err != nil {
		t.Fatalf("拿不到快照时不误报(另一条路会给失败): %v", err)
	}
}
