package host

// 取消语义的单测。
//
// 这一层是「用户点停止」能不能真的停下来的关键:取消必须体现在**下载循环**里
// (countingReader.Read),否则 SFTP 会一路把整个归档拉完,按钮就只是装饰。
//
// 另一个被钉住的不变量:取消后**不得**写入 .tdict-mirror.ok 基线标记 ——
// 那个标记是 MirrorReady() 的唯一判据,写下去就等于把半个镜像宣告成完整镜像,
// 下次增量拉取会基于残缺目录做增量,镜像永久缺文件。

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// 取消后 Read 立刻返回 context.Canceled,不再从底层读。
func TestCountingReaderStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 一上来就取消

	c := &countingReader{ctx: ctx, r: strings.NewReader("payload")}
	buf := make([]byte, 4)
	n, err := c.Read(buf)
	if n != 0 {
		t.Errorf("取消后仍读出了 %d 字节,应当一个字节也不读", n)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// 没取消时照常读;取消发生在**中途**也要立刻生效(不是只在开头判一次)。
func TestCountingReaderCancelsMidStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 一个很长的流:先读一次成功,取消后第二次必须失败
	c := &countingReader{ctx: ctx, r: strings.NewReader(strings.Repeat("x", 4096))}
	buf := make([]byte, 8)
	if _, err := c.Read(buf); err != nil {
		t.Fatalf("未取消时不该失败: %v", err)
	}
	cancel()
	if _, err := c.Read(buf); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消后 err = %v, want context.Canceled", err)
	}
}

// 进度回调仍然工作(取消逻辑不该把节流回调弄丢)。
func TestCountingReaderReportsProgress(t *testing.T) {
	var got []int64
	c := &countingReader{
		ctx: context.Background(),
		r:   strings.NewReader(strings.Repeat("y", 1024)),
		report: func(p MirrorProgress) {
			got = append(got, p.Bytes)
		},
	}
	if _, err := io.Copy(io.Discard, c); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[len(got)-1] != 1024 {
		t.Fatalf("进度回调 = %v, want 末次 1024", got)
	}
}

// 镜像打包阶段也认取消 —— 这条**不在这里测**:mirrorPackProgress 的取消是
// `select { <-ctx.Done() / <-ch }`,要真的驱动它就得有一个可被取消的 SSH 服务端,
// 而用一个 nil conn 去测会先在 goroutine 里空指针崩(取消与 goroutine 启动是竞态)。
// 取舍:不为了覆盖这一行去引一套假 SSH 服务;取消语义由上面 countingReader 的三条
// 用例钉住 —— 那才是"下载能不能真的停下"的关键路径,打包只是一次性等待。
