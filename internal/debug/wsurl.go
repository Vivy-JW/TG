package debug

// 服务测试（`/api/wstest`）本批已**从产品里取消**（仓库主人定性：它不是标准接口，
// 是临时测试工具 —— 所以是去掉端点，不是给它补闸门）。整个 `wstest.go` 连同
// `WSDefaultURLFor` / `WSTest` / `soapEnvelope` / `stripWSDLSuffix` / `wsEndpoints` 一起删了。
//
// **`wsURLPrefix` 必须留下**：它不是那条端点的私有件 —— `/api/status` 的 `wsPrefix`
// 字段（前端拼占位符地址用）就是它，与 `/api/wstest` 在不在无关。
// 单独抽到本文件，是为了让"这条规则只剩这一份实现"这件事在删掉 wstest.go 之后
// 仍然看得见（它的注释是从原文件逐字搬过来的，出处仍然准确）。

import "tt/internal/host"

// wsURLPrefix 是服务地址的前缀：出貨區(topprd)走 `/ws`，其余区走 `/w`。
//
// 单独抽出来是给 `/api/status` 用的（前端要拿它拼占位符地址）—— 这条规则只有这一份实现，
// 两边各写一遍的后果是"界面上的默认地址与真正发出去的地址不一致"（实测 zone=t 时
// 前端给 /wtopprd、后端给 /wstopprd，而这种偏差在本机 zone=36 时看不出来）。
//
// 注：从前这里还顺着 `WSDefaultURLFor`（服务测试默认地址）用过一次；那条端点已取消，
// 于是现在只有 `/api/status` 一处调用方。
func wsURLPrefix(zone string) string {
	if host.ZoneTNSName(zone) == "topprd" {
		return "/ws"
	}
	return "/w"
}
