package tzc

// `tt dev tzc uuid`：给"本次修改标记"生成一个 16 位十六进制 id。
//
// 为什么只到"生成 id"为止：标记与工号的**注释格式**（`#<单号>#<项次> … By <工号>`
// 与 `-s/-e` 块）是项目规范，不是包格式的一部分。把它塞进 apply 当硬闸门会让
// 合成包、老工作区、别人的流程一起被卡住。所以命令行只提供这一件无状态的小工具，
// 什么时候问、拒绝后写什么、块怎么包，全部由技能/提示词编排。
//
// 用法：tt dev tzc uuid [--json]  → 3f9a1c07b2d84e56

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"tt/internal/cli/dev/common"
)

// newModUUID 生成 16 位小写十六进制（8 字节随机数）。
// 抽成函数是为了让格式（长度、大小写、无分隔符）能被单测钉住。
func newModUUID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func cmdUUID(args []string) int {
	fs := flag.NewFlagSet("uuid", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc uuid [--json]")
		return 2
	}
	id, err := newModUUID()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	if *asJSON {
		common.EmitJSON(os.Stdout, map[string]any{"ok": true, "id": id})
		return 0
	}
	fmt.Println(id)
	return 0
}
