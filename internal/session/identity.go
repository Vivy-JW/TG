// Package session 是**会话身份**的唯一实现处：序号怎么分配、世代号怎么发。
//
// # 为什么要单独一个包
//
// 这两条规则本来住在 `internal/debug/manager.go` 与 `session.go` 里，和"连接池怎么维护"
// 混在一起。多会话之后，身份开始被第二处需要（命令要能说"这一次动作落在哪条会话上"），
// 于是必须有一个**不带调试器语义**的地方来放它们：
//
//   - 序号 `n`：**按环境**自增、取最小空号、会话消失即释放（C2.0 / Q16 / Q19）；
//   - 世代 `epoch`：进程内单调且唯一，复用/重启之后**绝不重复**（C5.3 / F15）。
//
// 本包只做这两件事，不认识 SSH、不认识 fgldb、不认识 GDC —— 所以两边都能用。
package session

import (
	"strconv"
	"sync/atomic"
)

// Slot 是分配器看到的"一条活会话"：只取它真正需要的三件事。
//
// 传结构体而不是 `*debug.Session`：本包不许 import 调试线（否则又变成一处耦合），
// 调用方把自己的类型摊平成这几个字段即可。
type Slot struct {
	Env    string // 环境名（序号的作用域）
	N      int    // 已占的序号（<=0 视为未分配，不占号）
	Exited bool   // 已断开：**不算占用**（Q19 的同源口径）
}

// NextSeq 分配某个环境下**最小的空号**。
//
// 语义（逐条对应契约，改之前先读）：
//   - 作用域是**环境**，不是企业、不是全局 —— 同机不同企业各算各的（Q16 / C2.2）；
//   - 取**最小空号**，不是"最大+1"：会话消失后它的号要能被后来者接手（C2.0 / Q19）；
//   - 已断开的残留**不占号**：它还在池子里只是等着被清扫，不该挡住新会话。
//
// 纯函数：不读全局状态、不加锁 —— 调用方在自己的锁里把 slots 拍好再进来。
func NextSeq(env string, slots []Slot) int {
	used := make(map[int]bool, len(slots))
	for _, s := range slots {
		if s.Exited || s.Env != env {
			continue
		}
		if s.N > 0 {
			used[s.N] = true
		}
	}
	for n := 1; ; n++ {
		if !used[n] {
			return n
		}
	}
}

// epochSeq 是进程内的世代计数（见 NewEpoch）。
var epochSeq atomic.Int64

// NewEpoch 发一个新的**会话世代号**（契约 C5.3 的 `epoch`，F15 的判据）。
//
// 语义：进程内**单调且唯一**的 base36 计数。
//
// 「唯一」这一条是刻意的：令牌不重复 ⇒ 复用/重启之后**不可能**凑巧等于调用方手上的旧值，
// 于是"旧句柄当场失效"（SESSION_STALE）不需要任何时钟比较或额外状态。
//
// 换代的三个时机（调用方负责，本包只管发号）：
//  1. 会话建立；
//  2. 同一宿主跑新一轮（复用空闲宿主 / 按日志重放复用宿主）；
//  3. 会话重启（收掉旧的 + 新的登上来，新会话自然拿到新的一代）。
func NewEpoch() string {
	return strconv.FormatInt(epochSeq.Add(1), 36)
}
