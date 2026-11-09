# internal/cli/dev — `tt dev` 命令组

三条线合一的接线层。目录即分层：

```text
dev/
├─ register.go / run.go   本包：cobra 树（路由、组级帮助、未知命令退 2、install 墓碑）
│                         + 根开关桥 takeRootFlags + 退出码透传 exitCode
├─ common/                两条线共用的脚手架：错误信封 Fail、稳定 JSON、ParseArgs、
│                         StripIdentitySuffix、config `tdev` 节接缝、总帮助 Usage
├─ tzc/                   .tzc 代码包线：八个动词核心 + uuid 生成 + 31 项对抗自检 → 见 tzc/README.md
└─ tzs/                   .tzs 表单包线：整线透传给引擎 → 见 tzs/README.md
```

## 依赖方向（没有环）

```text
本包 ──→ tzc ──→ common ←── tzs
  │      │                 ↑
  │      └──────→ tzs（仅 selftest：31 项对抗用例是 tdev 全集，覆盖两条线）
  └──────→ tzc, tzs（接线）
```

公共件为什么在 common 而不在任何一条线里：selftest 必然造成 tzc→tzs，
两条线又都要同一份输出契约 —— 公共件放任何一条线里，另一条就得反向依赖，成环。

## 参数解析权：不归 cobra

所有动词叶命令 `DisableFlagParsing` 原样透传，根命令的全局开关由 `takeRootFlags`
（run.go）从原始参数里翻译（`--config` → `TT_CONFIG`；`--json` 透传；
`--format json` → `--json`；`--csv`/`--env` **明确拒绝**并点名归属）。

为什么：参数写法契约冻结在 tdev 自己的解析器上（common.Usage："参数写法完全不变"，
位置无关、接受单杠长形式），pflag 接手会把 `-only` **静默错解析**成 `-o nly`。
cobra 在这条线上只负责路由与组级帮助。

## 退出码

各动词返回退出码，`exitCode`（实现根命令的 `alreadyReported` 接口）透传：
`0` 成功 / `2` 包格式或用法错 / `3` 验证失败 / `4` 写入被拒 / `5` IO·环境失败
（tzs 线没有 3，另有 1 = 引擎内部错）。**同一个数字在各线含义不同**，权威表见根命令
`--help` 与 [internal/cli/README.md](../README.md)。

## 判据

```bash
go test ./internal/cli/dev/... -count=1   # 三个包全绿（tzc 那一轮含 31 项对抗自检）
./tt.exe dev tzc selftest                 # 命令行口：通过 31，失败 0
```

`TestDevCobraRouting`（register_test.go，本包）直接驱动 cobra 树，钉住路由与退出码契约：
裸命令/未知动词/未知子命令退 2、动词 `--help` 退 0、`--csv`/`--env` 拒绝、`--config` 桥接。

**故意不覆盖**：动词核心的业务行为（那是 tzc/tzs 两包的判据），真实语料的形状分布
（那归 `TDEV_DEEP=1`）。

## 细节去哪

- `.tzc` 线动词、apply 管线、输出契约 → [tzc/README.md](tzc/README.md)
- `.tzs` 线命令面、引擎透传红线 → [tzs/README.md](tzs/README.md)
- 引擎客户端协议 → [../../dev/tzs/README.md](../../dev/tzs/README.md)
- 各动词的用法、常见错误、退出码速查 → [`skills/tt-dev-tzc/SKILL.md`](../../../skills/tt-dev-tzc/SKILL.md)
