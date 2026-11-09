# internal/str42s — `.42s` 编译字符串表

把 Genero/T100 编译出的二进制 `.42s` 读成 `token → 中文`。纯函数，零 IO；取文件与缓存由命令层负责。

## 文件布局

小端：

```text
uint32 count
uint32 hashes[count]    // 原工具的索引，本包不解释
uint32 offsets[count]
strings                 // 2*count 个 NUL 结尾的 UTF-8 字符串，逐对 token/value
```

字符串区固定从 `4 + 8*count` 开始。解析器只依赖 count 与字符串对，不猜哈希/偏移算法。

## 契约

- `Parse` 对截断、少对、多出非空尾巴、非法 UTF-8 都报错；
- 重复 token 保留在 `Entries`；`Lookup` 返回第一条，`LookupAll` 返回全部；
- `Unique` 对“重复但文本相同”判为不歧义，对“同 token 多个文本”返回 `ambiguous=true`；
- 不做表单前缀/后缀归一化；那是 [screen](../screen/README.md) 的合成规则。

## 判据

```bash
go test ./internal/str42s -count=1
```

合成夹具钉住布局、末尾 NUL 兼容、截断拒绝与重复 token 语义。真实 `.42s` 的样本验证由
`tt interface` 的集成/手工回读负责。

## 细节去哪

- 画面合成 → [../screen/README.md](../screen/README.md)
- 命令面 → [../cli/interfacecmd/README.md](../cli/interfacecmd/README.md)
