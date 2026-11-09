# internal/cli/outline — `tt outline`

把一个 `.4gl` 文件读成块树，供 Agent/人快速定位函数、UI 块和行范围。

```text
tt outline <file.4gl> [--human] [--flat] [--kind ...] [--depth N|all] [--brief] [--dir <目录>] [--at <环境>]
```

**做**：读本地文件或源码镜像；输出块树/扁平索引。

**不做**：不拨 SSH、不下载文件、不调用设计器、不支持 CSV、不拆 `show/list` 子命令。
解析本身复用 [internal/dev/fgl](../../dev/fgl) 的 `ParseOutline`，本包不复制正则。

## 数据源

优先级从高到低：

```text
位置参数 <file.4gl>    显式存在的文件
--dir <目录>           该环境镜像的最终目录（内含 4gl/）
--at <环境>            配置里该环境的镜像目录；--env 同义
```

`--dir` 与 `--at`/`--env` 不能同时给。没有默认环境；`--at` 只接受环境名，
带 `@企业` 会被 `--at` 的统一作用域拒绝。

显式文件存在时直接读；文件不存在时按位置参数继续找镜像：
裸文件名在镜像根下递归匹配，含路径的参数按相对路径找。同名文件命中多个
返回 `FILE_AMBIGUOUS`，并列候选，绝不替调用方挑一个。

## 输出

默认 JSON，信封：

```json
{
  "ok": true,
  "schema": "tt.outline/v1",
  "data": {
    "kind": "outline.tree",
    "file": "D:\\TT-Work\\109tst\\erp\\asf\\4gl\\demo.4gl",
    "source": {"mode": "mirror", "env": "109tst", "root": "D:\\TT-Work\\109tst"},
    "lineBase": 1,
    "total": 3,
    "returned": 3,
    "nodes": []
  }
}
```

节点字段：`label` / `kind` / `line` / `endLine` / `selStart` / `selEnd` / `children`。
行号 **1-based**；`selStart` / `selEnd` 是名称的 **0-based 字节列**。
叶子节点的 `children` 是 `[]`，不是 `null`。

`--brief` 去掉 `selStart` / `selEnd`；`--depth` 截断时树节点带
`childrenTruncated: true`，`data.truncated` 在树/扁平两种视图都有，
`returned` 是实际返回数，`total` 是解析出的总数。  
`--kind` 只保留命中的节点和通向它们的祖先路径，未命中的子树不输出；
合法值就是 `ParseOutline` 的九种 kind：

```text
FUNCTION MAIN REPORT DIALOG INPUT DISPLAY CONSTRUCT MENU SUB
```

`--flat` 把树变成数组，增加 `depth` 和 `parent`；`parent` 是同一返回数组里的下标，
不是跨调用的稳定 ID。`--human` 默认出树；`--human --flat` 出表格。
`--human --brief` 省略行号。

## 退出码

```text
0 成功（包括 --kind 过滤零命中，只要文件读到了）
2 用法错、--depth/--kind 非法、文件同名歧义（USAGE / ENUM_INVALID / FILE_AMBIGUOUS）
4 文件或镜像目录不存在（NOT_FOUND）
```

读文件失败用 `LOCAL_IO` 退 1；环境名不存在沿用统一码 `ENV_UNKNOWN` 退 2。

## 判据

```bash
go test ./internal/cli/outline -count=1
```

单测覆盖：数据源优先级、目录/相对路径查找、同名歧义、缺文件、非法扩展名、
kind 过滤保留祖先、depth 截断标记、flat 的全局 parent 下标、brief 去 selection、
默认 JSON/人类树/CSV 拒绝。

真实夹具回读：

```bash
go run . outline testdata/fgl-fixtures/input-on-key.4gl
go run . outline testdata/fgl-fixtures/input-on-key.4gl --human
go run . outline testdata/fgl-fixtures/input-on-key.4gl --flat --brief
go run . outline testdata/fgl-fixtures/input-on-key.4gl --depth 1
```

**本层负责证明**：给定文件/镜像能找到唯一文件，并把解析器的块树按契约投影；
**故意不覆盖**：4GL 词法与块边界本身（由 `internal/dev/fgl` 的 61 个夹具钉住），
以及真实镜像拉取（属于 `tt file mirror` 和集成测试）。

## 细节去哪

- 解析器与块类型 → [../../dev/fgl/README.md](../../dev/fgl/README.md)
- 镜像目录与拉取 → [../file/mirror.go](../file/mirror.go)
- 统一 JSON 信封 / 错误码 → [../../output/README.md](../../output/README.md)
