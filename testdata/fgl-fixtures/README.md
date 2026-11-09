# fgl-fixtures — 4GL 大纲夹具（Go 侧副本）

61 组 `<name>.4gl` + `<name>.expected.json`，给 `internal/dev/fgl` 的 4GL 大纲解析器做**文档级**
对拍：期望值来自语言文档，不是某个实现的输出快照。

- **消费者**：`internal/dev/fgl/outline_fixtures_test.go`（逐用例比对 `nodes` 与折叠区间）。
  实现本身在 `internal/dev/fgl/outline.go`。
- **来源**：BDL 项目（VS Code 扩展，MIT License，与本项目同作者）的 `test/fixtures/` 副本。
  溯源见 `README.source.md`。
- **期望值来源**：由 BDL 官方文档逐条推导，每个 `.expected.json` 的 `doc` / `about` 字段注明出处。
- **判据**：
  - 输出与 `nodes` 不等、用例带 `deviation` → 记为**已知偏差**，不算失败；
  - 不等且没有 `deviation` → **失败**。

  ```bash
  go test ./internal/dev/fgl -count=1 -v
  # 当前：用例 61 个，通过 58，已知偏差 3，不符 0
  ```

- **改动约定**：
  - 改了 `outline.go` 之后跑上面那条命令；**不符时先查实现**，不是先改期望。
  - 确实要改期望，必须同时补 `deviation` 说明（写明"文档上应当如此、实现已知未做到"）；
    **不许把实现输出回写当期望**。
  - 为与 BDL 保持 1:1 可比，**不要在本目录增删或修改用例**。

**同一套夹具有两份副本**：本目录（Go 侧）与 `web/app/scripts/fgl-fixtures/`（前端侧，供
`npm run check:outline` 使用）。两份的 `.4gl` / `.expected.json` 逐字节相同 —— **改一处要同步
另一处**，两份 README 各自维护自己那一侧的用法。
