# AGENTS.md

TT 是用于开发鼎捷数智旗下的大型ERP系统（T100）所构建的CLI工具。

## 改动纪律（GP 并入 TT）

任何涉及 GP / 设置页 / 调试工作台的改动，**动手前必须先把 T100 的对应实现找出来**，并说明：

1. T100 是怎么做的（文件、函数、关键行为）；
2. GP 现状与 T100 的差异；
3. 这次改动的必要性、影响面、可选方案；
4. 若 T100 没有对应路径，明确写“T100 无对应实现”，再说明为什么 GP 需要。

**默认等确认后再改**；除非用户明确说“直接改”。不要把“先加个锁/先加个重试”当成不需要对照 T100 的例外 —— 这类补丁同样要说明 T100 有没有、为什么 GP 需要。

## 项目结构

```
TT/
├─ AGENTS.md            仓库说明（AI/协作者必读）
├─ Makefile             必跑命令的快捷方式（make = build + test；深档目标自带副本闸门）
├─ config.example.json  配置文件样例（字段说明入口之一）
├─ VERSION              版本号唯一出处（两个打包脚本读它，make version-check 对照 web）
├─ build_portable.bat   便携包构建：前端 + Go 二进制 + skills/
├─ build_msi.bat        MSI 构建：复用便携载荷，生成 perUser 安装包
│
├─ .agents/             项目 Agent 技能与说明（不随发行包装到用户目录）
│  ├─ docs/             协作者共享文档与项目架构图
│  └─ skills/           项目内技能
│     ├─ asd-ste100/    技术英语/简化英语写作与检查
│     ├─ tt-add-cli/    新增 CLI 命令的设计指南
│     └─ tt-test/       真机集成测试与链路验证指南
│
├─ internal/            Go 后端全部实现
│  ├─ cli/              cobra 命令树装配处：各能力线 + 共用命令；唯一的接线处
│  │  ├─ at/            --at 文法、作用域与 --at-list 契约
│  │  ├─ common/        命令组间共享上下文（全局开关、MetaProvider、WebFS；叶子包防成环）
│  │  ├─ db/            tt db 命令组（sql 只读 SQL / ping 本机连库 / probe 服务器连库）
│  │  ├─ debug/         调试命令组门面（fgldb 那条线）

│  │  ├─ dict/          字典命令组：表/字段/校验/开窗/消息/参数/ACC 查询
│  │  ├─ file/          tg file 命令组：只读 SFTP（$TOP / $CUST 两棵树，根是实例目录）与离线镜像
│  │  ├─ interfacecmd/  tg interface 命令组：.42f + gae_file → 页面/字段/中文标签
│  │  ├─ outline/       tt outline 命令：.4gl → 函数/UI 块树
│  │  └─ session/       tt session 命令组：建/列/看/改 ENT/结束/收掉/透传/起作业
│  ├─ config/           统一配置层：位置解析、唯一写入口 Edit、原子写、打码、缓存清单
│  ├─ dbconfig/         配置里“库”那一节的连接模型
│  ├─ debug/            调试子系统：会话管理、断点存档、源码镜像 srccache、执行日志、企业快照
│  ├─ debugclient/      调试面 REST 唯一客户端（cli/debug 与 cli/session 共用；码/候选/信封）
│  ├─ designer/         从设计器 sites.xml 只读导入环境站点
│  ├─ dict/             字典子系统
│  │  ├─ db/            本地 SQLite 查询与 ACC 元模型/值查询
│  │  ├─ dbsync/        表与数据族同步（含 ACC 定义族）
│  │  ├─ live/          远程 ERP 直查（Oracle/金仓）
│  │  └─ server/        Web 字典接口
│  ├─ dev/              设计器包管线删剩的两个包（2026-11-11 删了 fence/verify/store/tzs/split/
│  │                     tapfile/tglfile/model/pkgfile/synth，形状库随 tzs 一起没了）
│  │  ├─ fgl/           FGL 语法/块/大纲处理（tt outline 在用，GP 的 4gl 也吃它）
│  │  └─ testutil/      语料根与遍历（CorpusRoot/CorpusFiles，测试夹具）
│  ├─ entdir/           企业目录（ENT→账号）快照、指纹、新鲜期（debug 与 dict 共用）
│  ├─ envmove/          环境改名的落点迁移（字典库/快照/镜像）与失败回滚（CLI 与 Web 共用）
│  ├─ erpdb/            客户端直连库封装（Oracle / 金仓，一个接口两个实现）
│  ├─ form42f/          .42f 编译表单解析：字段表 + 页面布局（纯函数，零 IO）
│  ├─ host/             唯一 SSH/远程服务器能力层（连接、登录、服务器侧只读探测）
│  ├─ output/           唯一输出出口（紧凑 JSON / CSV / 表格）
│  ├─ pathinstall/      用户 PATH（HKCU）增删，永不碰系统 PATH
│  ├─ safesql/          只读 SQL 文本层防线（远端查询入口用）
│  ├─ screen/           .42f 布局 + Labels 接口 → 前端画面模型
│  ├─ session/          会话身份规则（序号分配、世代号），不 import 调试线
│  ├─ skill/            技能（官方 + 第三方登记）：frontmatter 校验、冲突、assets 指针、安装
│  ├─ sshtun/           SSH 端口转发隧道（本地监听 → 远端 host:port）
│  ├─ target/           调试目标解析与候选信封
│  ├─ testenv/          本机测试环境解析（config.local.json + 环境变量，不带 testing）
│  ├─ testkit/          测试助手（stdout 捕获、吞输出、跳过台账、单源检查）
│  ├─ update/           tt 自身更新：查新版/下载校验/交棒给脱离进程（不自动更新）
│  ├─ web/              统一 Web 服务端：REST + WebSocket，嵌入 SPA，挂载调试/字典子系统
│  └─ winproc/          起/判/杀不随本进程死的子进程（Windows）
│
├─ web/                 前端（npm workspaces，根）
│  ├─ app/              唯一 workspace 成员：React + vite 调试工作台/设置页
│  │  ├─ src/           页面、组件、状态与 API 客户端
│  │  └─ scripts/       前端语料/校验脚本（含 fgl-fixtures）
│  ├─ shared/           共享主题与设计系统（被 @source 扫描，改动要谨慎）
│  └─ dist/             构建产物，被 Go 嵌入（.gitkeep 是承重墙）
│
├─ installer/           MSI 安装包定义（WiX v3；perUser 装到 %LOCALAPPDATA%\Programs\TT）
├─ tools/               构建辅助 Go 工具
│  ├─ zip/              跨平台确定性打包
│  └─ wixremovefolders/ MSI 卸载时补空目录清理
├─ testdata/            测试语料
│  └─ fgl-fixtures/     FGL 解析/写回夹具（Go 侧；前端侧有一份同内容副本）
└─ skills/              对外技能手册（评测中执行者唯一能读的东西）
   ├─ tg-erp-architecture/  TOP GP 整体架构与代码地图
   ├─ tg-erp-debug/     TOP GP 作业调试
   └─ tg-erp-read/      TOP GP 代码阅读（含 4gl / 42f）
```

## 开发命令

所有开发过程中用到的命令由[Makefile](Makefile)统一管理
