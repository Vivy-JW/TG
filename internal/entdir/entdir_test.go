package entdir

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 本包是"企业目录"的共享件：映射表、快照、环境指纹、新鲜期。
//
// 它的行为**已经**被 internal/debug 那侧间接跑到过 —— 那边有一层包装
// （srcmirror.go 的 pathSafeSeg → PathSafeSeg、ents.go 的 entFingerprint /
// entSnapshotPath / readEntSnapshot），debug 的测试测的是**那层接线**。
// 所以这里瞄准的是接线测不到的东西：本包自己的契约与边界（尤其是 Read 的每一条
// 拒绝理由、还有它**故意不判过期**这一点）。
//
// 快照是**可丢弃的缓存，不是数据源**：读坏了 / 写不出 / 指纹不符一律当没有，绝不因此报错。
// 这条性质是本包最要紧的契约，下面逐条钉。

func TestDBIdent(t *testing.T) {
	got := DBIdent("oracle", "db1", 1521, "t35prd")
	if want := "oracle|db1|1521|t35prd"; got != want {
		t.Errorf("得 %q，想 %q", got, want)
	}
	if got := DBIdent("", "", 0, ""); got != "||0|" {
		t.Errorf("空值也要给出稳定的形状，得 %q", got)
	}
}

// TestFingerprintChangesWithEveryComponent 四个组成部分**每一个**都必须影响指纹。
//
// 为什么不能少：zone 决定登录后的 T100 环境（31 开发 / 36 正式），而 oracle 的
// service 本身就带 zone 语义 —— 只按 host 做 key 会把开发区的企业清单当成正式区的。
func TestFingerprintChangesWithEveryComponent(t *testing.T) {
	base := Fingerprint("ssh1", "u1", "36", DBIdent("oracle", "db1", 1521, "t35prd"))
	muts := map[string]string{
		"sshHost": Fingerprint("ssh2", "u1", "36", DBIdent("oracle", "db1", 1521, "t35prd")),
		"sshUser": Fingerprint("ssh1", "u2", "36", DBIdent("oracle", "db1", 1521, "t35prd")),
		"zone":    Fingerprint("ssh1", "u1", "31", DBIdent("oracle", "db1", 1521, "t35prd")),
		"dbIdent": Fingerprint("ssh1", "u1", "36", DBIdent("oracle", "db1", 1522, "t35prd")),
	}
	for name, got := range muts {
		if got == base {
			t.Errorf("改了 %s 之后指纹没变 —— 换机器/换区域/换库会拿旧快照冒充", name)
		}
	}
	// 同样的输入必须给同样的指纹（否则每次读都当指纹不符，快照等于白存）。
	if again := Fingerprint("ssh1", "u1", "36", DBIdent("oracle", "db1", 1521, "t35prd")); again != base {
		t.Errorf("同一组输入该给同一个指纹：%q vs %q", again, base)
	}
}

// TestFingerprintEmptyDBIdentIsNone 没有库时那一格是字面量 "none"，不是空串 ——
// 空串会让 "oracle||" 与 "kingbase||" 之类的组合撞在一起。
func TestFingerprintEmptyDBIdentIsNone(t *testing.T) {
	got := Fingerprint("ssh1", "u1", "36", "")
	if want := "ssh1|u1|36|none"; got != want {
		t.Errorf("得 %q，想 %q", got, want)
	}
}

func TestEnvSeg(t *testing.T) {
	cases := []struct {
		name             string
		envName, host, z string
		want             string
	}{
		{"有环境名就用它", "示例测试区", "10.1.2.3", "36", "示例测试区"},
		{"环境名两侧空白剔掉", "  prd  ", "10.1.2.3", "36", "prd"},
		{"没有环境名退化成 主机-区域", "", "10.1.2.3", "36", "10.1.2.3-36"},
		{"没有区域就只留主机", "", "10.1.2.3", "", "10.1.2.3"},
		{"三次都空也要给个能用的段（不能是空串）", "", "", "", "_"},
		{"分隔符被压掉（不能逃出镜像根）", "../etc/passwd", "", "", ".._etc_passwd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EnvSeg(c.envName, c.host, c.z); got != c.want {
				t.Errorf("得 %q，想 %q", got, c.want)
			}
		})
	}
}

func TestPath(t *testing.T) {
	if got := Path("", "seg"); got != "" {
		t.Errorf("dataDir 为空该给空串（调用方据此跳过落盘），得 %q", got)
	}
	got := Path(filepath.Join("D:", "data"), "seg")
	if want := filepath.Join("D:", "data", "ents", "seg.json"); got != want {
		t.Errorf("得 %q，想 %q", got, want)
	}
}

func TestPathSafeSeg(t *testing.T) {
	cases := map[string]string{
		"asf":     "asf",
		"示例测试区":   "示例测试区", // 中文要保留：环境名可能是中文
		"a/b":     "a_b",
		"a\\b":    "a_b",
		"a:b*c?":  "a_b_c_",
		`a"b<c>d`: "a_b_c_d",
		"":        "_", // 空、当前目录、上级目录都不能作路径段
		".":       "_",
		"..":      "_",
		"  ..  ":  "_", // 剔空白之后还是 ".."，同样要挡
	}
	for in, want := range cases {
		if got := PathSafeSeg(in); got != want {
			t.Errorf("PathSafeSeg(%q) = %q，想 %q", in, got, want)
		}
	}
	// 挡掉 ".." 这件事值得单独断言一次：围栏里写的 ".." 不能变成往上跳一级。
	if strings.Contains(PathSafeSeg(".."), "..") {
		t.Error("PathSafeSeg 的输出里不该再出现 ..")
	}
}

// ---- 快照：Read / Write ----

func sampleSnapshot(fp string) *Snapshot {
	return &Snapshot{
		Version: Version, Env: "E1", Fingerprint: fp, Zone: "36",
		Dialect: "oracle", Target: "db1:1521/t35prd",
		Mappings:  []Mapping{{Ent: 99, Account: "ds"}, {Ent: 7, Account: "-", Placeholder: true}},
		FetchedAt: time.Now(),
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := Path(dir, "E1")
	fp := Fingerprint("ssh1", "u1", "36", "oracle|db1|1521|t35prd")

	if err := Write(p, sampleSnapshot(fp)); err != nil {
		t.Fatalf("写快照：%v", err)
	}
	s := Read(p, fp)
	if s == nil {
		t.Fatal("刚写下去的快照该读得回来")
	}
	if s.Env != "E1" || s.Zone != "36" || len(s.Mappings) != 2 {
		t.Errorf("读回来的内容不对：%+v", s)
	}
	// Placeholder 必须原样带回来 —— 它正是用来区分"没配账号"与"这个企业不存在"的。
	if !s.Mappings[1].Placeholder || s.Mappings[1].Account != "-" {
		t.Errorf("占位标记没保住：%+v", s.Mappings[1])
	}
}

// TestReadRejectsWhatItShould 逐条钉 Read 的**每一条拒绝理由**：
// 「快照是可丢弃的缓存」这句承诺全靠这四条。
func TestReadRejectsWhatItShould(t *testing.T) {
	dir := t.TempDir()
	fp := "ssh1|u1|36|none"
	good := Path(dir, "E1")

	write := func(t *testing.T, name string, body []byte) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("路径为空", func(t *testing.T) {
		if Read("", fp) != nil {
			t.Error("空路径该当没有")
		}
	})
	t.Run("文件不存在", func(t *testing.T) {
		if Read(filepath.Join(dir, "没有这个文件.json"), fp) != nil {
			t.Error("文件不在该当没有（不是报错）")
		}
	})
	t.Run("不是 JSON", func(t *testing.T) {
		if Read(write(t, "broken.json", []byte("{ 这不是 json")), fp) != nil {
			t.Error("坏文件该当没有")
		}
	})
	t.Run("版本不符", func(t *testing.T) {
		s := sampleSnapshot(fp)
		s.Version = Version + 1
		b, _ := json.Marshal(s)
		if Read(write(t, "badver.json", b), fp) != nil {
			t.Error("版本不符该当没有（不做兼容读）")
		}
	})
	t.Run("指纹不符", func(t *testing.T) {
		mustWrite(t, good, sampleSnapshot(fp))
		if Read(good, "另一台机器|u1|36|none") != nil {
			t.Error("指纹不符该当没有 —— 否则换区域会拿旧快照冒充")
		}
	})
	t.Run("映射表为空", func(t *testing.T) {
		s := sampleSnapshot(fp)
		s.Mappings = nil
		b, _ := json.Marshal(s)
		if Read(write(t, "nomap.json", b), fp) != nil {
			t.Error("空映射该当没有（那多半是写坏了，不是真没有企业）")
		}
	})
	t.Run("四条都对就读得出来", func(t *testing.T) {
		mustWrite(t, good, sampleSnapshot(fp))
		if Read(good, fp) == nil {
			t.Error("都对了还读不出来")
		}
	})
}

// TestReadDoesNotJudgeFreshness Read **不判过期** —— 这是刻意的，不是漏了。
//
// 过期快照在"现查失败"时是唯一的答案来源（服务器连不上、库连不上），
// 用不用由调用方按 TTL 自己定。在这里加一道过期判断，等于把那条退路堵死。
func TestReadDoesNotJudgeFreshness(t *testing.T) {
	dir := t.TempDir()
	fp := "ssh1|u1|36|none"
	p := Path(dir, "E1")
	s := sampleSnapshot(fp)
	s.FetchedAt = time.Now().Add(-100 * TTL) // 远超新鲜期
	mustWrite(t, p, s)

	if got := Read(p, fp); got == nil {
		t.Fatal("过期快照也该读得出来（判过期是调用方的事）")
	}
}

func TestWriteRefusesWithoutDataDir(t *testing.T) {
	if err := Write("", sampleSnapshot("x")); err == nil {
		t.Error("没有数据目录该报错（由调用方降级成一条提示，而不是静默丢弃）")
	}
}

// TestWriteIsAtomic 写失败不能留下"半个文件" —— 这里验的是正常路径下产物本身完好，
// 真正的原子性由 config.AtomicWrite 承担（那边有自己的测试）。
func TestWriteIsAtomic(t *testing.T) {
	dir := t.TempDir()
	p := Path(dir, "E1")
	if err := Write(p, sampleSnapshot("fp")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("落盘的不是合法 JSON（半个文件？）：%v", err)
	}
	// 目录里不该剩下临时文件（AtomicWrite 是"临时文件 → Rename"）。
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("落盘后目录里该只有快照本身，实得：%v", names)
	}
}

func mustWrite(t *testing.T, path string, s *Snapshot) {
	t.Helper()
	if err := Write(path, s); err != nil {
		t.Fatal(err)
	}
}
