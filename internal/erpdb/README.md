# internal/erpdb — 客户端直连库

客户端驱动的封装：一个接口、两个实现、一个按库类型分派的工厂。

| 实现 | 驱动 | 用于 |
|---|---|---|
| 金仓 | `pgx`（PostgreSQL 协议） | `type: kingbase` |
| Oracle | `go-ora`（纯 Go，无 cgo） | `type: oracle` |

## 契约

- `Connector` 只暴露四件事：类型、服务端版本、执行一条查询、关闭。
- 查询返回"列名 + 行"，**方言差异在实现里消化**（不要漏到调用方）。
- **没有绑定参数**：所有值经标识符白名单与字面量引号处理后内联。所以拼进语句的每一处
  都必须过 `ValidIdent` / `QuoteIdent` / `QuoteLit` —— 这是本包唯一的注入面。
- 执行前过一遍只读语句检查（`checkReadOnlyStatement` → `internal/safesql.Check`，
  **词法级真检查**）；**并且**在库侧强制只读 —— 见下面「两层只读防线」。两层都在代码里。

## 两层只读防线（本批补上的第二层）

**第一层：文本层**（`checkReadOnlyStatement`，`erpdb.go`）

把要去执行的语句交给 `internal/safesql.Check`（剥注释 / 词边界 / 词表 / 视图 / 包前缀），
nil 才放行。**挂在 `Query` 这唯一一个执行口上**（`ServerVersion` 除外，见下）。

> 从前这里是本包自己一份 `checkReadOnlySQL`：`TrimSpace`+`ToUpper` 看开头 + 看分号。
> 它挡得住"手写的 DML 直接贴进来"，**挡不住**前面加注释的写法（旧版 README 自己承认过），
> 更挡不住 `SELECT set_config('default_transaction_read_only','off',false)`。

**第二层：库侧**（两个实现各一处，都在 `Query` 这条路上）

| 库 | 落在哪 | 做了什么 |
|---|---|---|
| 金仓 | `OpenKingbase` 建池之前 | `poolCfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"` —— **会话级**，池里每条连接一建立就带上，服务器拒绝写（USERSET 级 GUC，普通账号即可设） |
| Oracle | `OracleConnector.queryReadOnly` | `BeginTx(ctx, nil)` → `SET TRANSACTION READ ONLY`（**必须是本事务第一条**）→ 查询 → `Rollback`；常规 DML 得 `ORA-01456` |

**这两层合起来才成立，缺一不可**（这不是"双保险"这种客套话，是分工）：

- 库侧那层**拦不住"把只读自己关掉"的语句** —— 金仓那条是会话级 GUC，一句
  `set_config('default_transaction_read_only','off',false)` 就能改回 off；
  挡住那一句的正是**第一层**（`set_config` 在 `safesql` 的禁用前缀表里，原话就是"直接拆掉只读事务"）；
- 文本层**拦不住"披着 SELECT 的写"**（自治事务、函数副作用、`EXECUTE IMMEDIATE`）—— 那要靠库侧事务。

**Oracle 那条比金仓多一次往返**（`SET TRANSACTION` 是真 RPC，与查询共用同一个 `ctx`），
而且 `database/sql` 的 `Tx` 会独占一条连接直到 `Rollback`（所以那个 `defer` 不是装饰：
漏了就是连接泄漏）。金仓那条对连接复用**零影响**（启动参数，连接建立时生效一次）。

**两处例外，如实写**：

- `ServerVersion`（`SELECT version()` / `SELECT banner FROM v$version …`）走的是**不带文本层**的
  执行体：两句都是**代码字面量**，不是调用方给的语句；而且 `v$version` 会被文本层当特权视图拒掉
  （那条规则本身是对的）。库侧只读照旧生效。
- **`ds` 账户的权限本仓库判不出来** —— 文档层面这两条语句都不要特殊权限
  （`SET TRANSACTION` 只要求"本事务第一条语句"，唯一账号级限制是**不支持 `SYS` 用户**；
  `default_transaction_read_only` 是 USERSET 级），但**"文档说不需要权限"≠"这个账号在这台库上一定行"**。
  必须真机验一次（同一条 `erpdb.Open` → `accounts[0]`，样例配置里就是 `ds`）：

  ```bash
  # Oracle：在能连到该库的机器上，用 ds 手工发一次
  sqlplus ds/<口令>@//<host>:<port>/<service> <<'EOF'
  SET TRANSACTION READ ONLY;
  SELECT 1 FROM DUAL;
  ROLLBACK;
  EOF
  # 期望：两条都成功 —— 没有 ORA-01456（只读事务里不允许它），更没有 ORA-01031: insufficient privileges

  # 金仓：同一台库上用 ksql 发一次
  ksql -h <host> -p <port> -U ds -d <库名> -c "SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY; SELECT 1;"
  # 期望：无 "ERROR: permission denied"

  # 补上本批改动之后，现成的判据是（它走的就是 ds，且不写库）：
  tt db ping --at <环境名>            # 期望仍打印 连接正常；若实例拒绝只读，这里会响亮失败
  ```

**挡不住的（不粉饰）**：自治事务 / 函数副作用（`safesql` 的包注释同款）；
Oracle 的 DDL 会隐式提交、从而**终结**只读事务；`EXECUTE IMMEDIATE` 这类"披着 SELECT 的写"。
**真正的边界是账号权限**（DBA 给 `ds` 的授权），工具层做不到。

## 入口

| 入口 | 作用 |
|---|---|
| `Open(ctx, conn)` | 按库类型分派出实现 |
| `Connector` | 接口（类型 / 版本 / 查询 / 关闭） |
| `SelectAllSQL(conn, table)` | 取全表的语句（表名过白名单） |
| `ValidIdent` / `QuoteIdent` / `QuoteLit` | 注入防线的三个原语 |

## 判据

```bash
go test ./internal/erpdb -count=1
```

**覆盖**的是**转义与闸门**那一层（安全关键，所以判据写得比"能跑"严）：
标识符白名单 `ValidIdent`（拒绝空格 / 分号 / 引号 / 点号 / 非 ASCII / 数字开头；
**不过滤保留字** —— 那是库的事）、`QuoteLit` 的单引号翻倍（含一组"想破墙"的载荷）、
`SelectAllSQL` 对用户名与表名的双重校验、
`checkReadOnlyStatement` 的**该拒的都拒 + 该放的都放**（含"前导注释绕过"与
`set_config` 那条**拆掉只读事务**的语句 —— 见
`TestCheckReadOnlyStatementBlocksTheReadOnlyTransactionBypass`）、
仓库里真实的字典查询字面量一条都不被误杀
（`TestRepoDictQueryLiteralsPassTheRealCheck` 扫 `internal/dict/live` 与 `internal/cli/dict`）、
两个 DSN 的拼法（oracle 走 URL 形式要转义 `% : @ / ?`，金仓走 key=value）、
`scanString` 把驱动给的任意值文本化。

**故意不覆盖**：真连库（`Open` / `OpenOracle` / `OpenKingbase` / 各连接器的 `Query`）——
属 L4，要真 Oracle 或金仓。**库侧那两层因此也没有离线判据**：金仓的那一行是连接参数
（要建池才生效）、Oracle 的那条是事务（要真库才拒绝写）。它们的判据是真机那两条命令
（见上面「两层只读防线」末尾），以及 `internal/debug` 不 import 本包那条静态检查
（`TestDebugPathDoesNotImportErpdb`：`tt debug sql` 走 SSH + sqlplus/ksql，
与本包的改动**没有交集**）。

**没有第二处文本闸门了**：本包从前那份前缀匹配已经删掉，`internal/safesql` 是全仓
唯一一份只读 SQL 文本检查（另一条通道 `internal/debug/dbsql.go` 用的也是它）。
`internal/dict/live` 那条路径自己构造 SQL（值走 `QuoteLit`、表名走 `ValidIdent`），
拼出来的语句同样要过这里这一道。

## 细节去哪

- 连接模型从哪来 → [../dbconfig/README.md](../dbconfig/README.md)
- 谁在用它、直查与本地镜像怎么选 → [../dict/live/README.md](../dict/live/README.md)
- 客户端到不了库时怎么走隧道 → [../sshtun/README.md](../sshtun/README.md)
