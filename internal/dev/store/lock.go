package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"
)

// lockStaleAfter 是锁的陈旧阈值：文件 mtime 超过它即视为上次进程已经死掉。
const lockStaleAfter = 30 * time.Minute

// lockRecord 是 .tdev/lock 的内容。
//
// 这是**唯一**允许出现时钟的地方（红线 R7：时间戳不是功能，只用于诊断锁的归属）。
type lockRecord struct {
	Pid     int    `json:"pid"`
	Host    string `json:"host"`
	Started string `json:"started"`
}

// Lock 取得工作区互斥锁，返回释放函数。
//
// 用 O_CREATE|O_EXCL 原子抢占 .tdev/lock（两个 tdev 进程不可能同时成功）。
// 已存在且**未陈旧** → *IOError（退出码 5）；已陈旧（mtime 超过 30 分钟）→
// 删掉再重试一次。
func (w *Workspace) Lock() (release func(), err error) {
	if w == nil || w.Dir == "" {
		return nil, &IOError{Msg: "Lock：工作区目录为空"}
	}
	if err := os.MkdirAll(w.tdevDir(), dirMode); err != nil {
		return nil, ioErr("创建 .tdev 目录失败 "+w.tdevDir(), err)
	}
	p := w.lockPath()

	err = w.tryCreateLock(p)
	if err == nil {
		return w.lockRelease(p), nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return nil, ioErr("创建工作区锁失败 "+p, err)
	}

	st, statErr := os.Stat(p)
	if statErr != nil {
		// 锁在这两步之间被别的进程释放了：再抢一次。
		if errors.Is(statErr, fs.ErrNotExist) && w.tryCreateLock(p) == nil {
			return w.lockRelease(p), nil
		}
		return nil, ioErr("检查工作区锁失败 "+p, statErr)
	}
	if time.Since(st.ModTime()) < lockStaleAfter {
		return nil, &IOError{Msg: fmt.Sprintf(
			"工作区已被另一个 tdev 进程锁定：%s 已存在（修改时间 %s）。"+
				"如确认没有 tdev 进程在运行，请手动删除该文件后重试。",
			p, st.ModTime().Format(time.RFC3339))}
	}
	// 陈旧锁：清理后重试一次。
	if rmErr := os.Remove(p); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		return nil, ioErr("清理陈旧锁失败 "+p, rmErr)
	}
	if err := w.tryCreateLock(p); err != nil {
		return nil, &IOError{Msg: fmt.Sprintf(
			"工作区已被另一个 tdev 进程锁定：%s 抢占失败（%v）。"+
				"如确认没有 tdev 进程在运行，请手动删除该文件后重试。", p, err)}
	}
	return w.lockRelease(p), nil
}

// tryCreateLock 用 O_EXCL 原子创建锁文件；已存在时返回 fs.ErrExist。
func (w *Workspace) tryCreateLock(p string) error {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	rec := lockRecord{Pid: os.Getpid(), Host: host, Started: time.Now().Format(time.RFC3339)}
	b, _ := json.Marshal(rec)
	b = append(b, '\n')

	_, werr := f.Write(b)
	serr := f.Sync()
	cerr := f.Close()
	if werr != nil || serr != nil || cerr != nil {
		_ = os.Remove(p)
		switch {
		case werr != nil:
			return werr
		case serr != nil:
			return serr
		default:
			return cerr
		}
	}
	return nil
}

// lockRelease 返回幂等的释放函数；删除锁文件时忽略「不存在」错误。
func (w *Workspace) lockRelease(p string) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			// 忽略 not-exist（锁已被陈旧判定清掉 / 已释放），也不因删除失败改变调用方结果。
			_ = os.Remove(p)
		})
	}
}
