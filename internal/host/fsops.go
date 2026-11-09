package host

// 远程文件的**只读**原语：列目录、看属性、读内容。
//
// 这是 `tt file` 的底座，也是“围栏”的第一层：本文件刻意只出现读操作
// （ReadDir/Lstat/Open/Read/ReadAt），**没有任何 Create/Remove/Rename/Mkdir/
// Chmod**。`tt file` 的源码守卫测试会盯住这一点（见 internal/cli/file/guard_test.go）。
//
// 为什么不用 shell：见 docs 里 `tt file` 的设计 —— 服务器侧不起进程，
// 只走 SFTP；命令语义（ls/grep/find/head…）在客户端实现。

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

// Entry 一个远程文件/目录的属性。
type Entry struct {
	Name    string      // 基名（目录列表里用）
	Path    string      // 完整路径（服务器路径，POSIX 分隔）
	Size    int64       // 字节数
	Mode    os.FileMode // 类型 + 权限位
	ModTime time.Time
	Link    string // 符号链接目标（仅 Lstat 且是链接时）
}

// IsDir 是不是目录（符号链接不算）。
func (e Entry) IsDir() bool { return e.Mode.IsDir() }

// IsSymlink 是不是符号链接。
func (e Entry) IsSymlink() bool { return e.Mode&os.ModeSymlink != 0 }

// ModeText 给出 `-rw-r--r--` / `drwxr-xr-x` 这种形态（Unix 的 10 位）。
//
// 用 os.FileMode.String() 不行：它用 Go 自己的字母表（符号链接是 `L`，不是 `l`），
// 而且多出来的位会打印成 `dalTLDpSugct?`，和人熟悉的 ls 输出不是一回事。
func (e Entry) ModeText() string { return ModeText(e.Mode) }

// ModeText 把权限位渲染成 ls 的 10 位形态。
func ModeText(m os.FileMode) string {
	t := byte('-')
	switch {
	case m.IsDir():
		t = 'd'
	case m&os.ModeSymlink != 0:
		t = 'l'
	case m&os.ModeNamedPipe != 0:
		t = 'p'
	case m&os.ModeSocket != 0:
		t = 's'
	case m&os.ModeDevice != 0 && m&os.ModeCharDevice != 0:
		t = 'c'
	case m&os.ModeDevice != 0:
		t = 'b'
	}
	bits := []os.FileMode{0o400, 0o200, 0o100, 0o040, 0o020, 0o010, 0o004, 0o002, 0o001}
	letters := "rwxrwxrwx"
	out := make([]byte, 10)
	out[0] = t
	for i, bit := range bits {
		if m&bit != 0 {
			out[i+1] = letters[i]
		} else {
			out[i+1] = '-'
		}
	}
	return string(out)
}

// FS 一条 SSH 连接上的只读文件视图。
//
// 只暴露读：想写就得改这个类型，而它没有写的方法 —— 这是“结构上只读”，
// 不是“约定只读”。
type FS struct {
	cl *sftp.Client
}

// FS 打开该连接的 SFTP 通道。调用方负责在结束时 Close。
func (c *SSHConn) FS() (*FS, error) {
	cl, err := c.SFTP()
	if err != nil {
		return nil, err
	}
	return &FS{cl: cl}, nil
}

// Close 关闭 SFTP 通道。
func (f *FS) Close() error { return f.cl.Close() }

// List 列一个目录（不含 . / ..，与 SFTP 一致）。结果按名字排序，输出稳定。
func (f *FS) List(dir string) ([]Entry, error) {
	infos, err := f.cl.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(infos))
	for _, fi := range infos {
		e := Entry{
			Name:    fi.Name(),
			Path:    path.Join(dir, fi.Name()),
			Size:    fi.Size(),
			Mode:    fi.Mode(),
			ModTime: fi.ModTime(),
		}
		// 符号链接目标只对链接有意义；读不到就让 Link 为空（不因此让整次列目录失败）
		if e.IsSymlink() {
			if target, err := f.cl.ReadLink(e.Path); err == nil {
				e.Link = target
			}
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Lstat 看一个路径本身（不跟随符号链接）。
func (f *FS) Lstat(p string) (*Entry, error) {
	fi, err := f.cl.Lstat(p)
	if err != nil {
		return nil, err
	}
	e := &Entry{
		Name:    path.Base(p),
		Path:    p,
		Size:    fi.Size(),
		Mode:    fi.Mode(),
		ModTime: fi.ModTime(),
	}
	if e.IsSymlink() {
		if target, err := f.cl.ReadLink(p); err == nil {
			e.Link = target
		}
	}
	return e, nil
}

// Stat 看一个路径的目标（跟随符号链接；由服务端解析）。
func (f *FS) Stat(p string) (*Entry, error) {
	fi, err := f.cl.Stat(p)
	if err != nil {
		return nil, err
	}
	return &Entry{
		Name:    path.Base(p),
		Path:    p,
		Size:    fi.Size(),
		Mode:    fi.Mode(),
		ModTime: fi.ModTime(),
	}, nil
}

// ReadLink 读符号链接目标。
func (f *FS) ReadLink(p string) (string, error) { return f.cl.ReadLink(p) }

// Open 打开一个文件用于读取（不允许写）。
func (f *FS) Open(p string) (*RemoteFile, error) {
	file, err := f.cl.Open(p)
	if err != nil {
		return nil, err
	}
	return &RemoteFile{file: file}, nil
}

// RemoteFile 只读的远程文件句柄。
type RemoteFile struct {
	file *sftp.File
}

// Read 实现 io.Reader。
func (r *RemoteFile) Read(p []byte) (int, error) { return r.file.Read(p) }

// ReadAt 实现 io.ReaderAt（tail 从文件尾部取行、随机读都靠它）。
func (r *RemoteFile) ReadAt(p []byte, off int64) (int, error) { return r.file.ReadAt(p, off) }

// Seek 实现 io.Seeker。
func (r *RemoteFile) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

// Stat 当前句柄对应的文件属性。
func (r *RemoteFile) Stat() (Entry, error) {
	fi, err := r.file.Stat()
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Name:    path.Base(fi.Name()),
		Size:    fi.Size(),
		Mode:    fi.Mode(),
		ModTime: fi.ModTime(),
	}, nil
}

// Close 关闭句柄。
func (r *RemoteFile) Close() error { return r.file.Close() }

// copyN 是 io.Copy 的限额版本：最多读 limit 字节，返回实际拷贝字节数与“是否被截断”。
//
// 给 get 用（--max）；limit<=0 表示不限。
func CopyN(dst io.Writer, src io.Reader, limit int64) (int64, bool, error) {
	if limit <= 0 {
		n, err := io.Copy(dst, src)
		return n, false, err
	}
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return n, false, err
	}
	if n > limit {
		return limit, true, nil
	}
	return n, false, nil
}

// NotFound 把 SFTP 的“文件不存在”判断集中到一处（不同实现的文案不统一）。
func NotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
		return true
	}
	// pkg/sftp 的 StatusError 在不同调用里不一定实现 Is/Unwrap；
	// 兜住它自己的文案（"no such file" / "does not exist"）。
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "does not exist")
}

// WrapPathErr 把底层错误补上路径（SFTP 的报错常常只说“no such file”）。
func WrapPathErr(p string, err error) error {
	if err == nil {
		return nil
	}
	if NotFound(err) {
		return fmt.Errorf("不存在: %s", p)
	}
	return fmt.Errorf("%s: %w", p, err)
}
