# fgl-fixtures — 4GL 大纲夹具（前端副本）

61 组 `<name>.4gl` + `<name>.expected.json`，给 `web/app/src/fgloutline.ts`（移植自 BDL 的
4GL 大纲解析器）做**文档级**对拍：期望值来自语言文档，不是某个实现的输出快照。

- **消费者**：`web/app/scripts/fgloutline.fixtures.mjs`，由 `npm run check:outline` 驱动。
- **来源**：BDL 项目（VS Code 扩展，MIT License，与本项目同作者）的 `test/fixtures/` 副本。
- **期望值来源**：由 BDL 官方文档逐条推导，每个 `.expected.json` 的 `doc` / `about` 字段注明出处。
- **判据**：输出与 `nodes` 不等、用例带 `deviation` → **已知偏差**（不算失败）；不等且没有
  `deviation` → **失败**。`deviation` 声明的是"文档上应当如此、实现已知未做到"，
  例如 `SUBDIALOG` 不建节点、`;` 不作终止符、`record.*` 的标签丢 `.*`。

  ```bash
  cd web && npm run check:outline --workspace app   # 只跑这一项
  cd web && npm run check:app                       # 三项一起跑
  ```

  运行器还支持 `--strict`（把已知偏差也算失败）、`--filter <子串>`、`--list`、`--tree`。

- **改动约定**：
  - 改了 `web/app/src/fgloutline.ts` 之后跑上面那条命令；**不符时先查实现**。
  - 确实要改期望，必须同时补 `deviation` 说明；**不许把实现输出回写当期望**。
  - 为与 BDL 保持 1:1 可比，**不要在本目录增删用例**；高亮相关的对抗用例写在
    `scripts/fgltokens.test.mjs` 里，别混进来。

**同一套夹具有两份副本**：本目录（前端侧）与 `testdata/fgl-fixtures/`（Go 侧，供
`go test ./internal/dev/fgl` 使用）。两份的 `.4gl` / `.expected.json` 逐字节相同 ——
**改一处要同步另一处**。

重新同步（若上游 BDL 侧更新了夹具）：

```powershell
Copy-Item D:\我的项目\BDL\test\fixtures\*.4gl            web\app\scripts\fgl-fixtures\ -Force
Copy-Item D:\我的项目\BDL\test\fixtures\*.expected.json  web\app\scripts\fgl-fixtures\ -Force
# 同步完把 testdata\fgl-fixtures\ 下同名文件也覆盖一遍，两份必须一致
```
