# tzs-mini —— 仓库自带的**最小 .tzs 工作区**

一套 `.tzs` 语料夹具：**三个典型表单包 + 它们需要的最小元数据，3.2 MB，进仓库**。

它解决的是这个：`.tzs` 的语料回归从前只有「有真客户语料的那台机器」能跑 ——
一份完整工作区 **130 MB**，而且 `mta/` + `tbl/` 是数据不是代码，不该入库。
于是干净克隆上那几条回归**全跳**。有了这份夹具，clone + `cd engine && ./build.sh` 之后就能跑。

## 里面是什么

```
ws/                         ← 工作区（引擎的 TZSCLI_WS 指这一层）
  mta/          19 个文件    全量减去 zooms.xml / subroutines.xml / messages.xml
  <模块>/tbl/   26 个 .tbl   只拷被那三个包引用到的表
  cpmp530(c).tzs
  aapp320(c).tzs
  aapt300(c).tzs
manifest.txt                ← 45 行的固定清单（sha256-16 + 字节 + 仓库相对路径）
```

生成器在 [tools/tzsmini](../../tools/tzsmini)（2026-10 从本目录的 `build.py` 迁去），
重新生成 `ws/` 与 `manifest.txt` 两样。

三个包是**按"小 / 中 / 大 + 特征"挑的**，不是随手抓的：

| 包 | 规格节点 | 节点种类 | 控件种类 | 标签页 | 动作 | 容器 | 特点 |
|---|---|---|---|---|---|---|---|
| `cpmp530(c).tzs` | 12 | 3 | 18 | 1 | 1 | 9 | 最小；且是自带 `posX` 漂移那三个包之一（stale=2） |
| `aapp320(c).tzs` | 168 | 5 | 21 | 3 | 1 | 26 | 中等；有标签页 |
| `aapt300(c).tzs` | 901 | 6（含 `tree`） | 26 | 12 | 26 | 65 | 最大；Tree + 多页 + 多动作 |

实测这三个包在这份夹具上与在**真语料工作区**里**逐字段一致**（`SUMMARY` 的四个计数、规格节点数、
布局元素数、`stale`）。这一条是生成时验过的，也是 `manifest.txt` 要钉住的东西。

## 判据

```bash
go test ./internal/dev/tzs -run TestMiniCorpus -count=1 -v
# 期望四条全 PASS：
#   TestMiniCorpusRoundTrip    读得进、存出来还是同一个定点（四个零 + stale 等于 pristine）
#   TestMiniCorpusWritePath    往包里加字段
#   TestMiniCorpusEditPath     改 spec 属性并回读
#   TestMiniCorpusIsPinned     夹具的字节没变
```

前三条**不需要任何环境变量**（语料在仓库里），只需要引擎 —— 引擎不在仓库里（构建产物，
见 [engine/BUILD.md](../../engine/BUILD.md)），缺了就整条跳过。
第四条**连引擎都不需要**，所以它在任何机器上都守着夹具。

重新生成：

```bash
go run ./tools/tzsmini --src <一个真工作区>
# --src 必须是工作区（该目录下有 mta/），包摊在它顶层
```

## 故意不覆盖什么

- **只三个包**。它能证明「这条链路没坏」，**证明不了**「上百个真实包的形状分布还都对」。
  那一条仍然归 `TTZS_DEEP` 的真语料回归。
  这份夹具是**冒烟网**，不是回归网。
- **`mta/` 少了三份目录性数据**（`zooms.xml` 消息目录 / `subroutines.xml` 子程序 / `messages.xml`）。
  实测不影响这三个包的任何计数，但**若将来有测试要查这三样，看到的是一个更小的世界**。
- **不是字节级的"完整工作区"**。它是从一份真工作区**筛**出来的，不是压缩包。

## 改它的约定

**改夹具 = 改判据**。前三条回归的基线是从这份夹具**自己**量的（`requireFixedPoint` 拿 pristine
当基线），所以一份悄悄变了的夹具会让判据跟着一起搬而**全是绿的**。实测过一次：把 `mta/`
裁到只剩四个文件时，引擎照样回 `SUMMARY|ok`，而 `aapt300` 的布局元素已经从 515 变成了 512。

所以：**任何对 `ws/` 的改动都必须连 `manifest.txt` 一起重新生成**（`tools/tzsmini` 一次做两件事）。
`TestMiniCorpusIsPinned` 就是这么盯着的 —— 多一个、少一个、改一个字节都红。

## 两个路径陷阱（生成脚本里也写着）

- **工作区字符串必须用反斜杠**。设计器会归一化**包路径**、不会归一化工作区字符串：
  `TZSCLI_WS=C:/ws` + 包任意形式一律 `NotInCurrentWorkspaceException`；包路径用正斜杠没关系。
  完整规则见 [skills/tt-dev-tzs/SKILL.md](../../skills/tt-dev-tzs/SKILL.md) §4.4。
- **包必须在工作区目录之下**（纯字符串前缀比较），所以别把包拷到 `%TEMP%` 再喂给引擎。

## 细节去哪

- 生成规则、为什么留哪些文件 → [tools/tzsmini/main.go](../../tools/tzsmini/main.go) 的文件头（每条都注明是实测）
- 那三条回归本身 → [internal/dev/tzs/mini_corpus_test.go](../../internal/dev/tzs/mini_corpus_test.go)
