package tzs

// doctor.go —— 环境自检：把「为什么这条命令用不了」在**一条命令里**答完。
//
// 这条命令存在的理由是排查成本。tzs 这条链上有五个独立的失败面，而它们的现象几乎一样：
//
//	exe 路径不对        → 「问不到管道名」
//	设计器没装/装错版本  → 「冷启动 60 s 未就绪」（真正的错误藏在守护进程日志里）
//	工作区没配/配错     → 「没有守护进程在监听」（因为管道名是按工作区算的）
//	有陈旧构建的孤儿     → 一切正常，只是在吃内存
//	状态文件损坏        → 已被 loadState 静默挪走，但用户该知道
//
// 它们的**共同现象**都是「连不上」，所以照着现象排查会一路走错。doctor 把它们分开答，
// 每一条都给出「我在哪看到的」和「怎么办」。它不 Boot 任何东西（只跑两个不 Boot 的开关），
// 所以可以随便跑。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/testenv"
	"tt/internal/winproc"
)

// 自检条目的三个等级。
const (
	LevelOK   = "ok"
	LevelWarn = "warn" // 能用，但有一处值得知道的事
	LevelFail = "fail" // 这么下去命令一定失败
)

// Check 是一条自检结果。
type Check struct {
	Name    string `json:"name"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// DoctorReport 是自检报告（命令层负责渲染成文本或 --json）。
type DoctorReport struct {
	Checks []Check `json:"checks"`
}

// OK 报告有没有 fail 级条目（warn 不影响退出码）。
func (r *DoctorReport) OK() bool {
	for _, c := range r.Checks {
		if c.Level == LevelFail {
			return false
		}
	}
	return true
}

// Failed 返回 fail 级条目的名字（命令层报错时列出来）。
func (r *DoctorReport) Failed() []string {
	var out []string
	for _, c := range r.Checks {
		if c.Level == LevelFail {
			out = append(out, c.Name)
		}
	}
	return out
}

func (r *DoctorReport) String() string {
	var b strings.Builder
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "  [%-4s] %-14s %s\n", c.Level, c.Name, c.Message)
	}
	return b.String()
}

// add 记一条。签名收**成品文案**而不是 format：一处 %s 打错的格式串会把一条
// 「文件不存在」的提示变成 `%!s(MISSING)`，而这种错只有真的走到那一支才看得见。
func (r *DoctorReport) add(level, name, msg string) {
	r.Checks = append(r.Checks, Check{Name: name, Level: level, Message: msg})
}

// Doctor 跑一遍环境自检。它**总是**返回一份报告（每条失败都记在报告里），
// 因为「哪几条通过、哪几条没过」本身就是答案；用 error 表达只会让人只看到第一条。
func Doctor(ctx context.Context, o Options) *DoctorReport {
	r := &DoctorReport{}

	// ① 引擎 exe：存在 → 能答 --manifest。
	if strings.TrimSpace(o.Exe) == "" {
		r.add(LevelFail, "引擎 exe", "没有指定路径（--exe / 配置）")
	} else if _, err := os.Stat(o.Exe); err != nil {
		// 两种处境给两种话。源码树下的人看到的路径是 <仓库>\tzs\...，那条路径本来就
		// 不该存在 —— 把他也说成"装坏了"会让人去重装一个根本没错的东西。
		r.add(LevelFail, "引擎 exe", fmt.Sprintf("%s 不存在或读不到（%v）\n"+
			"        装的是发行包 → 包不完整，重装或重新打包；\n"+
			"        从源码跑 → 引擎不在包里，先 cd engine && ./build.sh，"+
			"再把 tzs.serverExe 指到 engine/out/tzs-server.exe", o.Exe, err))
	} else if m, err := FetchManifest(ctx, o.Exe); err != nil {
		r.add(LevelFail, "引擎 exe", fmt.Sprintf("%s 答不了 --manifest：%v", o.Exe, err))
	} else {
		slow := 0
		for _, f := range m.Fns {
			if f.Slow {
				slow++
			}
		}
		r.add(LevelOK, "引擎 exe", fmt.Sprintf("%s（函数表 %d 个，慢函数 %d 个）", o.Exe, len(m.Fns), slow))
	}

	// ② 设计器程序集：默认是**随包分发**的那份（<引擎 exe 目录>\designer），TZSCLI_INSTALL
	// 只作开发期覆盖。这里刻意不再有"没配"这一种状态 —— 设计器目录不是配置项了。
	d := strings.TrimSpace(o.InstallDir)
	bundled := d == ""
	if bundled {
		d = filepath.Join(filepath.Dir(o.Exe), "designer")
	}
	if st, err := os.Stat(d); err != nil || !st.IsDir() {
		// fail 而不是 warn：少了它 Boot 一定失败，而现象是「冷启动 60 s 未就绪」——
		// 一条与真正原因毫无关系的消息。
		//
		// 消息分两种处境，因为"目录不在"有两个完全不同的原因：发行包缺件（打包坏了），
		// 与源码树本来就没有随包的那份（引擎在 engine/out/，不是包）。只说前者会让后者
		// 去重装一个根本没错的东西。
		if bundled {
			r.add(LevelFail, "设计器目录", fmt.Sprintf("%s 不是目录（%v）\n"+
				"        装的是发行包 → 包不完整，重装或重新打包；\n"+
				"        从源码跑 → 先 cd engine && ./build.sh，"+
				"它会把仓库里的 engine/designer/ 采到 out/designer/", d, err))
		} else {
			r.add(LevelFail, "设计器目录",
				fmt.Sprintf("TZSCLI_INSTALL 指向的 %s 不是目录（%v）；改指到正确的路径，或取消这个环境变量", d, err))
		}
	} else {
		var missing []string
		for _, f := range []string{"SpecDesignerCommon.dll", "SpecDesigner.FormEditor.dll"} {
			if _, err := os.Stat(filepath.Join(d, f)); err != nil {
				missing = append(missing, f)
			}
		}
		if len(missing) > 0 {
			r.add(LevelFail, "设计器目录", fmt.Sprintf("%s 里缺 %s", d, strings.Join(missing, ", ")))
		} else {
			src := "随包分发"
			if !bundled {
				src = "TZSCLI_INSTALL 覆盖"
			}
			r.add(LevelOK, "设计器目录", fmt.Sprintf("%s（%s）", d, src))
		}
	}

	// ③ 工作区：配置了吗 → 是目录吗 → 像不像设计器工作区。
	ws, werr := o.workspace()
	// 报出来源：工作区按环境配（设置页「站点管理 → 环境 → 工作区目录」），也可能来自
	// --workspace 或 TZSCLI_WS —— 引擎开在别的客户的目录上时，报错本身看不出异常。
	src := ""
	if o.WorkspaceSrc != "" {
		src = "（来自 " + o.WorkspaceSrc + "）"
	}
	switch {
	case werr != nil:
		r.add(LevelFail, "工作区", "未配置（--workspace / TZSCLI_WS / --env <环境名> 的工作区 三选一；"+
			"环境那一份在设置页的「站点管理 → 环境 → 工作区目录」）")
	default:
		st, err := os.Stat(ws)
		switch {
		case err != nil || !st.IsDir():
			r.add(LevelFail, "工作区", fmt.Sprintf("%s%s 不是目录：%v", ws, src, err))
		default:
			if mst, merr := os.Stat(filepath.Join(ws, "mta")); merr != nil || !mst.IsDir() {
				r.add(LevelWarn, "工作区",
					fmt.Sprintf("%s%s 里没有 mta/（可能不是设计器工作区；不是的话 open 会在加载阶段失败）", ws, src))
			} else {
				r.add(LevelOK, "工作区", ws+src)
			}
		}
	}

	// ④ 管道名：能问出来就说明 exe + 工作区两件事都对了。
	if werr == nil {
		if pipe, err := PipeName(ctx, o.Exe, ws); err != nil {
			r.add(LevelWarn, "管道名", fmt.Sprintf("问不到：%v", err))
		} else {
			r.add(LevelOK, "管道名", pipe+"（含 MVID：重编引擎就变，所以每次都问）")
		}
	}

	// ⑤ 状态文件与守护进程。
	dir := o.StateDir()
	if st, err := LoadState(dir); err != nil {
		r.add(LevelWarn, "状态文件", fmt.Sprintf("%s 读不动（下次写会重建）：%v", StatePath(dir), err))
	} else if len(st.Daemons) == 0 {
		r.add(LevelOK, "状态文件", StatePath(dir)+"（还没有记录）")
	} else {
		var parts []string
		for _, k := range st.Keys() {
			e := st.Daemons[k]
			state := "已死"
			if e.PID > 0 && winproc.Alive(e.PID) {
				state = "活着"
			}
			parts = append(parts, fmt.Sprintf("%s pid=%d %s", k, e.PID, state))
		}
		r.add(LevelOK, "状态文件", StatePath(dir)+"："+strings.Join(parts, "；"))
	}
	if werr == nil {
		if info, err := LookupDaemon(ctx, o); err != nil {
			r.add(LevelWarn, "守护进程", err.Error())
		} else if info.Running {
			r.add(LevelOK, "守护进程", fmt.Sprintf("在跑（pid %d，管道 %s）", info.PID, info.Pipe))
		} else {
			r.add(LevelOK, "守护进程", "没在跑（下一条命令会自己起来）")
		}
	}

	// ⑥ 语料根：**只有深档语料回归需要它**，所以是 warn 而不是 fail ——
	// 少了它 `tt dev tzs <动词>` 照常能用；缺的是"跑不了那一批回归"，不是"命令会失败"。
	// 用 fail 会把每个没语料的人的 doctor 变成红的。
	//
	// "这次是怎么定的、卡在哪一处"由 testenv 说了算（CorpusRootDetail）—— 这里自己猜的话
	// 会把指引指到错的那条路上（比如明明是环境变量指错了，却说"你没设环境变量"）。
	if root := testenv.CorpusRoot(); root != "" {
		r.add(LevelOK, "语料根", fmt.Sprintf("%s（来自 %s）", root, testenv.CorpusRootDetail()))
	} else {
		r.add(LevelWarn, "语料根", fmt.Sprintf(
			"没找到 —— %s。只有 TTZS_DEEP=1 / TDEV_DEEP=1 的深档语料回归需要它",
			testenv.CorpusRootDetail()))
	}

	return r
}
