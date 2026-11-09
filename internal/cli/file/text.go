package file

// 文本类动词（cat/head/tail）的实现：读文件、切行、二进制拒绝。

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/host"
)

const binaryProbe = 8 << 10 // 前 8KB 里有 NUL 就按二进制处理

// isBinary 用“前 8KB 是否含 NUL”判断二进制（与 git 的做法一致）。
func isBinary(b []byte) bool { return bytes.IndexByte(b, 0) >= 0 }

// openChecked 打开一个文件并做二进制探测；返回的 reader 已包含探测时读掉的前缀。
//
// 只读一次探测块：open → ReadAt(0, 8KB) → 重新 Seek(0)，后续读取不受影响。
func openChecked(p string) (*host.RemoteFile, []byte, error) {
	f, err := cur.fs.Open(p)
	if err != nil {
		return nil, nil, err
	}
	buf := make([]byte, binaryProbe)
	n, err := f.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		f.Close()
		return nil, nil, err
	}
	prefix := buf[:n]
	if !allowBinary && isBinary(prefix) {
		f.Close()
		return nil, nil, usageErr("二进制文件（含 NUL 字节），不往终端喷：%s", display(cur.top, p)).
			WithHint("要落盘用 tt file get <路径> -o <本地文件>；确要输出原始字节加 --human --binary（或 tt file get <路径> -o -）")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, prefix, nil
}

// headLines 从 r 读前 n 行（不把整个文件读进内存）。
func headLines(r io.Reader, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	out := make([]string, 0, n)
	for sc.Scan() {
		out = append(out, sc.Text())
		if len(out) >= n {
			break
		}
	}
	return out, sc.Err()
}

// tailLines 从 ra 的尾部往前取 n 行。
//
// 只读尾部若干块（最多 256KB）直到凑够 n+1 个换行，避免把大文件整份读进来。
func tailLines(ra io.ReaderAt, size int64, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	const chunk = 64 << 10
	const maxRead = 4 << 20
	var data []byte
	offset := size
	for offset > 0 && len(data) < maxRead {
		step := int64(chunk)
		if step > offset {
			step = offset
		}
		offset -= step
		buf := make([]byte, step)
		if _, err := ra.ReadAt(buf, offset); err != nil && err != io.EOF {
			return nil, err
		}
		data = append(buf, data...)
		if bytes.Count(data, []byte{'\n'}) > n {
			break
		}
	}
	lines := strings.Split(string(data), "\n")
	// 文件以换行结尾时，最后一个元素是空串，去掉
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// readTextLines 取一个文件的前/后 n 行（head/tail 共用）。
func readTextLines(f *host.RemoteFile, useTail bool, n int) ([]string, error) {
	if useTail {
		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		return tailLines(f, st.Size, n)
	}
	return headLines(f, n)
}

// runText 执行 head/tail：多个文件时按 GNU 的 `==> 文件 <==` 分段。
func runText(cmd *cobra.Command, args []string, useTail bool, n int) error {
	if len(args) == 0 {
		return usageErr("%s 需要至少一个文件", cmd.CommandPath())
	}
	var files []string
	for _, arg := range args {
		matches, err := cur.expandArg(arg)
		if err != nil {
			return err
		}
		files = append(files, matches...)
	}
	type lineFile struct {
		Path  string   `json:"path"`
		Lines []string `json:"lines"`
	}
	out := make([]lineFile, 0, len(files))
	for _, arg := range files {
		p, err := cur.argPath(arg)
		if err != nil {
			return err
		}
		p, err = cur.resolveSafe(p)
		if err != nil {
			return err
		}
		self, err := cur.fs.Lstat(p)
		if err != nil {
			if host.NotFound(err) {
				return notFoundErr(cur.top, p)
			}
			return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
		}
		if self.IsDir() {
			return usageErr("%s 是目录：%s", cmd.CommandPath(), display(cur.top, p))
		}
		f, _, err := openChecked(p)
		if err != nil {
			return err
		}
		lines, readErr := readTextLines(f, useTail, n)
		f.Close()
		if readErr != nil {
			return ioErr("%v", host.WrapPathErr(display(cur.top, p), readErr))
		}
		out = append(out, lineFile{Path: display(cur.top, p), Lines: lines})
	}
	if wantJSON(cmd) {
		return commonPrintJSON(map[string]any{"ok": true, "files": out})
	}
	for i, f := range out {
		if len(out) > 1 {
			if i > 0 {
				fmt.Fprintln(os.Stdout)
			}
			fmt.Fprintln(os.Stdout, "==> "+f.Path+" <==")
		}
		for _, ln := range f.Lines {
			fmt.Fprintln(os.Stdout, ln)
		}
	}
	return nil
}
