// Package testenv 是**本机测试环境**的配置：一份 config.local.json + 环境变量覆盖。
//
// 为什么单独一个包，而不是塞进 internal/dev/testutil 或 internal/testkit：
// **读它的代码不能带 `testing`**。`internal/dev/testutil` 要用它来定语料根，而 testutil 被
// **生产代码** import（internal/cli/dev/selftest.go，`tt dev tzc selftest` 的实现），
// 一旦那条链上出现 `testing`，整个 `testing` 包就会进 `tt` 二进制。于是分层是：
//
//	testenv（本包：配置与解析，无 testing）
//	   ↑                    ↑
//	testutil（夹具 + 语料文件遍历）   testkit（*testing.T 那层包装：跳过还是判失败）
//
// 配置**只放机器路径，不放口令**。口令在 config.json 的 hosts.sshs[].db 里，那是**凭据**；
// 测试配置是"这台机器上引擎在哪、语料在哪"。混在一起等于让测试配置继承凭据的处理规矩
// ，而 config.json 本身是用户在用的东西。
//
// 优先级一律是 **环境变量 > config.local.json > 内置缺省**，与
// internal/dev/testutil/corpus.go 已经立下的"谁先设谁说话"同构。
// **覆盖项设了却指不到目录时返回空，不回落到缺省** —— 那表示调用方指错了地方，
// 静默换一份别的去用，会留下"我以为跑的是这份"这种错误结论。
package testenv
