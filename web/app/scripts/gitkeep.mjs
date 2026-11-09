// 构建后把 web/dist/.gitkeep 写回去。
//
// 为什么需要它：vite 的 emptyOutDir 会清空整个 dist 目录，包括这个占位文件 ——
// 而它**必须存在于仓库里**：main.go 的 `//go:embed all:web/dist` 在该目录不存在时
// 直接构建失败（pattern all:web/dist: no matching files found），于是全新克隆下来
// 连 `go build` 都过不去。
//
// 只靠 .gitignore 的 `!/web/dist/.gitkeep` 是不够的：文件被构建删掉之后就不在工作区，
// `git add` 无从添加，改完当场看不出问题、下次克隆才炸。所以让构建负责恢复它。
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const dist = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', 'dist')
fs.mkdirSync(dist, { recursive: true })
fs.writeFileSync(
  path.join(dist, '.gitkeep'),
  '# 前端构建产物占位（npm run build 会清空本目录并重新生成，构建后由 postbuild 写回本文件）。\n' +
  '# 它必须存在于仓库里：main.go 的 //go:embed all:web/dist 在该目录不存在时构建失败，\n' +
  '# 于是全新克隆连 go build 都过不去。\n',
)
