# `testdata/form42f/` —— `.42f` 解析的**合成小夹具**

依据：`.agents/plan/tt-genero-formdef-contract.md` 的 **C7.2 / C7.3**（Q6 定案）。

**这里只放合成数据**：字段名一律 `demo*`，**不含任何真实业务数据**（真样本走 `TT_FORM_SAMPLE_DIR`，
默认指向 `.agents/Genero Original Factory Compatible/fixtures/forms/`）。

| 文件 | 用途 | 守哪条 |
|---|---|---|
| `basic.xml` | 正常表单：2 个 `FormField` + 1 个 `TableColumn` + `RecordView`/`Link` + `LStr` | C1.4 / C1.5 |
| `tablecolumn_only.xml` | **只有** `TableColumn` / `PhantomColumn`，没有 `FormField` | **C1.3 反证**：若实现里过滤掉 `TableColumn`，这条必红 |
| `duplicate_colname.xml` | 同表单内两个条目**重名** `demo201`（fieldId 5 / 6） | C1.5（进 `Duplicates`，不覆盖） |
| `missing_attrs.xml` | 缺 `colName` / 缺 `fieldId` / `fieldId` 非数字，共 3 条应被跳过 | C1.6（`Skipped` 计数） |
| `not_xml.txt` | 纯文本 | C1.7（明确报错） |
| `empty.xml` | 空内容 | C1.7（明确报错） |
