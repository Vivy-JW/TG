# internal/dev/tglfile — `.tgl` 里的标记

`.tgl` 是设计器的**框架骨架**。本包只做一件事：在里面找到并操作三类标记。

| 标记 | 是什么 | 谁用 |
|---|---|---|
| **区段边界**（`{<section>}`） | 框架区段的起止，带 id | 合成时与 `.tap` 的 `<section>` 配对；改区段时两边必须一致 |
| **占位符**（裸名插入点） | 框架预留的插入位置 | 合成时与 `.tap` 的点配对；TGL 有占位符而 TAP 无点 → 造空点 |
| **集合锚点**（`other.function` / `other.dialog` / `other.report`） | 自订定义点被注入的位置 | 新增点只能插进这类锚点 |

## 入口

| 函数 | 作用 |
|---|---|
| `TrimEnd` | 读入时按设计器的习惯去掉尾部空白 |
| `FindSections` / `SectionBody` | 找区段并取出正文 |
| `FindPlaceholders` | 找占位符（含它是哪一类） |
| `FindAnchor(kind)` / `ReplaceAnchor(kind, repl)` | 找/替换集合锚点（新增自订点走这里） |
| `AnchorSectionID(prog, id)` | 判断某个区段 id 是不是锚点区段 |
| `PatchSection(tgl, id, content, eol)` | 按区段打补丁（改区段时同步 `.tgl`） |

## 关系

- 被 [synth](../synth/README.md)（占位符 ↔ 点配对）、[fence](../fence/README.md)、
  [verify](../verify/README.md)、[split](../split/README.md)（打补丁）、[cli](../../cli/dev/README.md) 使用。
- **改区段要写两处**：`.tap` 的 `<section>` 与 `.tgl` 的同一区段必须逐字节一致 ——
  这条约束由 [split](../split/README.md) 落地、由 `selftest` 里的对抗用例盯着。

## 判据

```bash
go test ./internal/dev/tglfile -count=1
```

**覆盖**：`TrimEnd` 去掉**所有**尾部空白（含 CR/LF/VT/FF）、区段配对（数量不等 → 格式错，
退出码 2，且详情里报两个计数）、`readonly="Y"` 的大小写不敏感、占位符的**大小写敏感**
（与区段标记相反，这是设计器的真实差别）、锚点正则里那个 `.` 是**任意字符**
（`other_function` / `otherXfunction` 也命中 —— 照抄设计器）、`PatchSection` 的补丁格式
（`标记 + eol + 正文 + eol + 标记`，eol 由调用方给）。

还有一条**注释与代码不符**的实证：`ReplaceAnchor` 的注释说用 `ReplaceAllLiteral` 防
`$` 展开，而代码调的是 `ReplaceAll`（**会**展开）。实践上不出事（repl 里不会有 `$`），
但注释承诺的那层保护不存在 —— `TestReplaceAnchorExpandsDollarInRepl` 钉的是**实际行为**。

**故意不覆盖**：设置器的真实行为（那要装设计器）。这里的判据来源是反编译源码里的位置，
都逐条写在了包注释里。

## 细节去哪

- 三类标记怎么与 TAP 配对、配对不上各是什么后果 → [../synth/README.md](../synth/README.md)、[../verify/README.md](../verify/README.md)
- 三方依据（四个正则原文与锚点展开逻辑的出处） → 设计器反编译源码 [`designer-src/`](../../../designer-src/README.md) 的 `CodeEditorManager.cs`
