package config

import (
	"path/filepath"
	"testing"
)

// TestSkeletonHasNoDesignerDir 钉住两件事：设计器目录与工作区都不是机器级配置项。
//
// 设计器的程序集现在随包分发（见 engine/src/Designer/Bootstrap.cs 的 Install，默认是
// `<引擎自己的目录>\designer`），所以 config.json 里不该再出现 installDir。
// 工作区按**环境**配（hosts.sshs[].workspace）—— 骨架里没有环境，所以也不该有它。
//
// 这两条都是防“顺手加回来”的：installDir 回来会让同一份 tt 在两台机器上跑两版设计器；
// 而机器级 workspace 回来会让“切了环境就该换工作区”这件事再次失去意义，
// 而现象是引擎开在另一个客户的目录上、报错里却看不出来。
func TestSkeletonHasNoDesignerDir(t *testing.T) {
	sk := NewSkeleton()
	sec, ok := sk["tzs"].(map[string]any)
	if !ok {
		t.Fatalf("骨架里没有 tzs 节：%#v", sk["tzs"])
	}
	for _, bad := range []string{"installDir", "workspace"} {
		if _, ok := sec[bad]; ok {
			t.Errorf("tzs 节里不该有 %s（设计器随包分发；工作区按环境配）", bad)
		}
	}
	if _, ok := sec["serverExe"]; !ok {
		t.Error("tzs 节少了 serverExe")
	}
}

// TestTzsStatusDesignerFollowsEngine 钉住 Designer 的位置规则：跟着**生效的**引擎 exe 走。
//
// 引擎自己就是这么算的（`<自己所在目录>\designer`）。两边各算一次的话，设置页会报"有"
// 而引擎报"没有" —— 那是最难查的一类不一致。
func TestTzsStatusDesignerFollowsEngine(t *testing.T) {
	def := filepath.Join(t.TempDir(), "tzs", "tzs-server.exe")

	got := TzsStatusOf(TzsSettings{}, "", def).Designer.Dir
	if want := AbsPath(filepath.Join(filepath.Dir(def), "designer")); got != want {
		t.Errorf("Designer 缺省位置 = %q，想要 %q", got, want)
	}

	// serverExe 覆盖之后，Designer 要挪到覆盖后那份的旁边，而不是留在缺省旁边。
	other := filepath.Join(t.TempDir(), "elsewhere", "tzs-server.exe")
	got = TzsStatusOf(TzsSettings{ServerExe: other}, "", def).Designer.Dir
	if want := AbsPath(filepath.Join(filepath.Dir(other), "designer")); got != want {
		t.Errorf("覆盖 serverExe 后 Designer = %q，想要 %q", got, want)
	}

	// 引擎 exe 都算不出来时不硬凑一个路径（DirStatus 空 = 未配置）。
	if got := TzsStatusOf(TzsSettings{}, "", "").Designer.Dir; got != "" {
		t.Errorf("没有引擎 exe 时 Designer 该为空，得到 %q", got)
	}

	// 工作区**按环境**配（NamedSsh.Workspace），这里拿到的是调用方给的那一份；
	// 没给就是未配置（不再有机器级的 tzs.workspace 兜底）。
	ws := filepath.Join(t.TempDir(), "ws")
	if got := TzsStatusOf(TzsSettings{}, ws, def).Workspace.Dir; got != AbsPath(ws) {
		t.Errorf("工作区该取传进来的当前环境那一份，得到 %q（想要 %q）", got, AbsPath(ws))
	}
	if got := TzsStatusOf(TzsSettings{}, "", def).Workspace.Dir; got != "" {
		t.Errorf("没给工作区时该是未配置，得到 %q", got)
	}
}
