# internal/dev/pkgfile — 包的只读视图与字节级重建

`.tzc` / `.tzf` / `.tzx` 包（zip 容器）的**只读视图**，以及写回的唯一实现。
`.tzc` 管线的每一步都以这里拿到的 `*Package` 为输入。

## 契约

- **`Package` 不可变**。只读，没有 `SetPoint` / `SetTgl` 之类的接口；写回一律
  `Build(Rebuild{...})`，产出**新字节**（红线 R5）。
- **`Plan` 不改动任何字节**，只算逐条目写回计划 —— 它是 `--dry-run` 与 apply 报告的数据源：
  每条目给 `Changed` / `Reason` / 新旧大小 / 新旧 sha256 / `Method` / `Passthrough`。
- **红线**：条目集合不增不减、名字不变、顺序不变（R2，设计器打包时只遍历已有条目）；
  `.4gl` 永不改写（R1，它是服务器产出）。

## 包类型与必需条目

类型**由扩展名决定**，不是猜内容。定下类型后按设计器的清单检查必需条目，缺一个就是包格式错
（退出码 2）：

| 类型 | 扩展名 | 必需条目 |
|---|---|---|
| 报表 | `.tzr` | `.tap` `.tgl` |
| 代码 | `.tzc` | `.tap` `.tgl`（差异包另加 `.src` `.apt`） |
| 代码规格 | `.tzx` | `.csd` |
| 报表规格 | `.tzf`（ReportSpec） | `.rsd` |
| 表单 | `.tzs` | `.tsd` `.4fd`（简单表单另加 `.4fdref`） |
| 报表代码 | `.tzg` | ——（设计器自身没有该分支，不检查） |

## 版本文件

版本条目按**名字**取，只读**第一行**，解析成 `Major.Minor[.Build[.Rev]]`。
比较时**只比 Major / Minor**（补丁号不同不算不匹配）；`OpenOptions.AcceptVer` 默认 `"1.0"`。

## 为什么写回是"字节级重建"

设计器（.NET）写出来的包有它自己的形态：**局部头里没有数据描述符**（标志位 bit 3 = 0），
CRC / 压缩后大小 / 原大小直接写在局部头里，每个条目还带自己的 extra 字段
（真机实测是两段：时间戳与打包器标记）。

而通用的 zip 写入器会**无条件**置 bit 3、补一个数据描述符，还会用自己的时间戳 extra 顶掉原有
extra。这样产出的包在设计器里打不开 —— 它会报找不到数据描述符签名。

所以写回不是"重新写一个合法 zip"，而是**照抄设计器的写法**，逐字节重建：

```
· 局部头与中央目录记录：从原包逐字节拷贝，只打补丁
    清 bit 3、写真实 CRC/大小、更新局部头偏移、必要时更新 Zip64 extra 的 8 字节字段
· 未改动的条目：连压缩数据都逐字节照抄
· 一切 extra 字段：原样保留（不新增、不删除、不重排）
· 版本号 / 时间 / 外部属性 / 条目注释：不动
```

于是**零改动重建与原包逐字节相同**，这是本包最重要的判据。
压缩方法与修改时间沿用原条目，**不取时钟**（红线 R7）。被改写的条目用**它原来的压缩方法**
重新压；方法无法复现时回退 deflate，并在 `EntryAction.Reason` 里说明。

## 解析机制

| 环节 | 做法 | 理由 |
|---|---|---|
| 找 EOCD | 从尾部往前找，条目注释最长按 64 KiB 处理 | 与通用实现一致 |
| 定位条目 | **以中央目录为准**，局部头只用来取原始字节与 extra | 中央目录是权威索引 |
| 一致性校验 | 局部头与中央目录里的**名字必须一致**，不一致直接判包格式错 | 写得出来也读不回来 |
| Zip64 | 按官方附录的固定顺序读/写（原大小 → 压缩后大小 → 局部头偏移 → 磁盘号）；只有被写成 `0xFFFFFFFF` 的 32 位字段才在 extra 里出现 | 标记位决定要读写几个字段 |
| Zip64 回写 | **只在条目被改写、或它前面的条目变长导致偏移变化时**才动 | 未改动的条目不许"顺手修正" |

## 导出入口

| 入口 | 用途 |
|---|---|
| `Open(path, OpenOptions)` / `Package` / `Entry` / `Kind` | 打开并检视包 |
| `Package.Tap() / Tgl() / Full4gl()` | 取三类具名条目 |
| `Package.Plan(Rebuild)` / `Build(Rebuild)` | 算计划 / 产出新字节 |
| `Rebuild{Tap, Tgl, Replace}` | 这次替换什么（其余条目保持原字节） |
| `ParseVersion` / `Version` | 版本条目解析与比较 |
| `UnzipTo(src, dst, force)` | `.tzs export` 用的纯解压（另一条线） |
| `FormatError` / `IOError` / `ExitCodeError` | 错误类型，各自带退出码（2 / 5） |

## 判据

```bash
go test ./internal/dev/pkgfile -count=1
# TestRebuildIdentityByteExact   零改动重建与原包逐字节相同
# TestRawParseMatchesStdlib      自研解析与标准库在语料条目上逐字段一致
# TestRoundtripZeroChangeCorpus  语料级 roundtrip：逐条目 sha256 一致
# TestBuildRedlines              R1（.4gl 不改）/ R2（条目集合不变）
# TestKindFromExt / TestParseVersion / TestFormatErrorExitCode
```

`zipshape_test.go` 另外盯着 zip 形态本身（位标志、extra 保留、Zip64 字段）。

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| zip 重建逻辑 | 几乎整条管线都依赖 `Build()`。`TestRebuildIdentityByteExact` 与 `TestRoundtripZeroChangeCorpus` 必跑；**gate3 只能验"能装载"，验不出"设计器会不会拒绝这个容器"** —— 所以形态规则要照抄，不要"优化" |
| 必需条目清单 | 与设计器的分支一一对应，直接改变 `Open` 的拒绝面（退出码 2），并影响两条件测试包（`export` 对不完整包的处理） |
| `Rebuild` / `EntryAction` 字段 | `--dry-run` 与 apply 报告的输出形状；改字段名等于改对外契约 |
| 压缩方法回退策略 | 被改写条目的字节。判据始终是**逐条目内容 sha256**，不是整体字节相等 |

红线 R1（`.4gl` 不改）/ R2（条目集合不变）/ R5（不可变）都在这一层落地。

## 细节去哪

- zip 原始结构的读写实现 → `zipraw.go`（顶部注释是这套规则的完整说明）
- 写回之后的事（拆分与落盘） → [../split/README.md](../split/README.md)、[../store/README.md](../store/README.md)
- 三方依据 → 设计器反编译源码 [`designer-src/`](../../../designer-src/README.md)：容器层、条目→加载器分派、打包
