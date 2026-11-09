package debug

// probe 的"验了什么"清单与收尾判据（纯逻辑：不碰网络、不打印、不读会话）。
//
// # 为什么要把判据抽出来
//
// `tt debug probe` 的现场是"真 SSH + 真 fgldb"，整条命令进不了单测 —— 于是它成了一条
// **没人守的自检**，而它上一版的毛病正是这样长出来的：
//
//	默认作业 `bsft001_wf` 在 main 之后要开界面，画面没接线时当场 `Fatal error -6300` 退出
//	（A35，真机三次复现）。probe 把"程序自己跑完"当成错误，于是**前六步全绿也照样 exit 1**。
//
// 一条恒失败的自检比没有自检更费时间：人看到失败会去查 SSH、查断点，而真因是"这作业跑完了"。
// 这正是 `probe_test.go` 文件头那句老话的同一个毛病，只是换了个形式。
//
// # 判据
//
// 一句话：**前六步必须每一步都真的走通**；步进与求值因为程序跑完而没走到，不算失败，
// 但必须如实报"未覆盖"（不许拿它当成功糊过去，也不许省掉不提）。
//
// 分成 required / optional 两档而不是"有几步算几步"，是因为这两档的**后果完全不同**：
// required 缺一步 = 链路故障，要修；optional 缺一步 = 这个作业活不到那一步，链路没问题。

import (
	"fmt"
	"strings"

	"tt/internal/debug"
)

const (
	probeLogin  = "登录并启动"
	probeEntry  = "入口停站"
	probeBreak  = "break main"
	probeRun    = "run 命中"
	probeSource = "源码镜像"
	probeWhere  = "where 调用栈"

	probeStep  = "next 步进"
	probePrint = "print 求值"
)

// probeRequired 是链路本身：缺一步就是故障，与"程序跑完了"无关。
//
// `probeSource` / `probeWhere` 也算在内：取不到停站处的源码或栈帧，说明"停站之后那两条读取"
// 没通（源码目录未知那类故障正是这样表现）。**空列表不算验过** —— 那是"看起来正常"的假象。
var probeRequired = []string{probeLogin, probeEntry, probeBreak, probeRun, probeSource, probeWhere}

// probeOptional 是依赖"程序还活着"的两步：程序自己跑完时它们必然走不到。
var probeOptional = []string{probeStep, probePrint}

// probeVerdict 判一次 probe 的结果，返回是否通过 + 没走通的步骤（按上面两档的固定顺序）。
//
// `reached` 必须是**真实走通**的步骤，由调用方在每一步确实成功后追加 —— 这里不做任何补全，
// 也不把"没出错"当"验过了"。判据只有一条：required 全部出现。
func probeVerdict(reached []string) (ok bool, missing []string) {
	got := make(map[string]bool, len(reached))
	for _, s := range reached {
		got[s] = true
	}
	ok = true
	for _, s := range probeRequired {
		if !got[s] {
			ok = false
			missing = append(missing, s)
		}
	}
	for _, s := range probeOptional {
		if !got[s] {
			missing = append(missing, s)
		}
	}
	return ok, missing
}

// isProbeRequired 判某个步骤是不是链路本身（决定收尾清单里打 ✓ 还是 ✗）。
func isProbeRequired(step string) bool {
	for _, s := range probeRequired {
		if s == step {
			return true
		}
	}
	return false
}

// probeProgramExited 判一次命令失败是不是"程序自己跑完退出"（而不是链路故障）。
//
// 两个判据同时看，理由不同：
//   - **错误文本**：会话层读到 `Program exited` 时用 `"program exited"` 收口本轮命令
//     （`internal/debug/session.go` 的 `reProgramExited` → `completePending`）；
//   - **会话状态**：同一条路径随后调 `enterIdle`（"程序已退出…会话保留"），所以状态会落到
//     idle / exit。文本可能被上游改写，状态兜一次。
//
// 两者都不会把真正的失败（登录失败、断点没设上、run 不停站）算成正向 —— 那些路径上
// 错误文本不是这句，状态也不会是 idle。
func probeProgramExited(err error, st debug.State) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "program exited") {
		return true
	}
	return st == debug.StateIdle || st == debug.StateExit
}

// printProbeVerdict 把收尾判定打成一份"验了什么 / 没验什么"的清单。
//
// 为什么要单独打印这份清单：probe 的价值就是"我说通了，你看得见通在哪"。只给一句
// "完成"等于让人自己去猜覆盖范围；而"程序跑完导致没验到"与"链路坏了"长得一模一样，
// 必须在这里分开写。
func printProbeVerdict(reached, missing []string, ok bool, exitAt string) {
	fmt.Println("\n[probe] 结果:")
	for _, s := range reached {
		fmt.Printf("  ✓ %s\n", s)
	}
	for _, s := range missing {
		if isProbeRequired(s) {
			fmt.Printf("  ✗ %s  没验到（链路故障）\n", s)
			continue
		}
		why := "未覆盖"
		if exitAt != "" {
			// 退出发生的**那一步**与它后面的步骤要分开说：前者是"就在这一步退的"，
			// 后者是"还没走到这里"。一律写成"之前"会把 next 自己也说错。
			if s == exitAt {
				why = "未覆盖（就在「" + exitAt + "」这一步，程序自己跑完退出，不是链路故障）"
			} else {
				why = "未覆盖（程序在「" + exitAt + "」这一步自己跑完退出，没走到这里）"
			}
		}
		fmt.Printf("  - %s  %s\n", s, why)
	}
	if ok {
		fmt.Println("  ⇒ 链路通过：登录/启动/停站/断点/命中/源码/栈帧都真的走通了")
		return
	}
	fmt.Println("  ⇒ 不通过：上面的 ✗ 项是链路故障，不是程序跑完")
}
