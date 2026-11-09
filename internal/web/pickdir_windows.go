//go:build windows

package web

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// pickDirPS 用 Windows Shell 的 BrowseForFolder 弹原生目录选择框。
// 路径通过环境变量传进去，不拼进命令文本 —— 初始目录里带引号/分号也不会变成脚本。
//
// 三个刻意的写法：
//
//  1. **不用 System.Windows.Forms**。原实现是 `FolderBrowserDialog` + 现场 `Add-Type` 编译
//     C#（只为隐藏控制台窗口）—— 每次点按钮要加载 WinForms 程序集再调一次 csc，秒级。
//     现在走 Shell.Application 的 BrowseForFolder：不加载 WinForms、不编译、也不弹控制台，
//     并且它**原生接受一个初始路径**。
//  2. **初值不存在时上溯到存在的父目录**。BrowseForFolder 只在第 4 参数是个真实目录时才
//     定位过去；配的是"还没建的目录"时旧实现直接不传，对话框就停在上次位置 —— 看起来就是
//     "没置顶"。上溯之后至少会打开到它附近。
//  3. **显式把输出设成 UTF-8**。路径含中文时按控制台默认代码页输出会变成乱码，
//     而 Go 侧是按 UTF-8 读的。
const pickDirPS = `
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$initial = $env:TT_PICK_INITIAL
if ($initial -and -not (Test-Path -LiteralPath $initial -PathType Container)) {
  $p = $initial
  while ($p) {
    $parent = Split-Path -LiteralPath $p -Parent
    if (-not $parent -or $parent -eq $p) { $p = ''; break }
    if (Test-Path -LiteralPath $parent -PathType Container) { $p = $parent; break }
    $p = $parent
  }
  $initial = $p
}
$title = if ($env:TT_PICK_TITLE) { $env:TT_PICK_TITLE } else { '选择文件夹' }
$shell = New-Object -ComObject Shell.Application
$f = $null
if ($initial) {
  # 第 4 参数在部分系统上不接受 BSTR,失败就退回"不带初位置",不要让整个选择器报错。
  try { $f = $shell.BrowseForFolder(0, $title, 0, $initial) } catch { $f = $null }
}
if (-not $f) { $f = $shell.BrowseForFolder(0, $title, 0) }
if ($f) {
  $picked = $f.Self.Path
  if ($picked) { [Console]::Out.Write($picked) }
}
`

func pickDirectory(ctx context.Context, initial, title string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-NoProfile", "-STA", "-WindowStyle", "Hidden", "-Command", pickDirPS)
	cmd.Env = append(os.Environ(),
		"TT_PICK_INITIAL="+initial,
		"TT_PICK_TITLE="+title,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", false, ctx.Err()
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", false, fmt.Errorf("打开文件夹选择器失败: %s", msg)
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", true, nil // 用户取消：PowerShell 正常退出但没有输出
	}
	return path, false, nil
}
