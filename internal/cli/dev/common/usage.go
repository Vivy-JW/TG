// Usage 是 tdev 两条线（tzc + tzs）的**总帮助**与退出码契约文本。
//
// 放在 common 而不是任何一条线里：父包 register.go 的 devLong 拼它（裸 `tt dev`
// 打它、退 2），用法文本漂移测试（TestUsageTextHasNoStaleAdvice）扫它 ——
// 谁都可以引，谁也不是它的主人。
package common

// Usage 是总帮助。
const Usage = `tt dev —— T100 设计器包工具：.tzc 代码包 + .tzs 表单包

用法：
  tt dev tzc export <pkg.tzc> [-o <dir>] [--only <点名>...] [--json]
        # -o 省略时默认导出到 <包所在目录>/<程序名>-ws
        # （身份后缀 (c)/(s) 会去掉：D:\pkg\capt110(c).tzc → D:\pkg\capt110-ws\）
  tt dev tzc status [<dir>] [--json]
  tt dev tzc verify [<dir>] [--json] [--strict]
  tt dev tzc apply  [<dir>] [-o <pkg.tzc>] [--dry-run] [--yes] [--json]
  tt dev tzc unlock [<dir>] [--yes] [--json]     # 框架解锁（单向状态迁移，只改 workspace）
  tt dev tzc rename [<dir>] <旧函数名> <新函数名> [--scope …] [--desc …]  # 结构事务（只改 workspace）
  tt dev tzc newfn  [<dir>] --type FUNCTION|DIALOG|REPORT [--name …]      # 新增自订点（只改 workspace）
  tt dev tzc selftest [--json]

  tt dev tzs export <pkg.tzs> [-o <dir>] [--force] [--json]
        # 表单包**纯解压**（.tzs / .tzv）：不解围栏、不校验、不产生工作区
        # -o 省略时默认解压到 <包所在目录>/<程序名>-unzip
        # 产物是**只读参考**（没有 tzs apply）；要改表单走下面的 tzs 动词
  tt dev tzs <动词> --args '<JSON 对象>' [--form <程序名>] [--args-file <文件>] [--workspace <dir>] [--rpc-timeout <秒>] [--json]
        # 读写表单，**唯一**写路径 —— 由设计器自己的引擎算，不是我们拼 XML
        # 55 个动词由引擎的函数表生成（open / form_tree / set_spec_attr / set_spec_attrs /
        # nudge / validate / save / close …），所以没有"函数名"这一层要填
        # 参数**只用 JSON 给**；用 --form 指定是哪张已打开的表单（不必搬运句柄）
        # 例：tt dev tzs open          --args '{"path":"D:\\pkg\\aapp320(c).tzs"}' --json
        #     tt dev tzs nudge         --form aapp320 --args '{"paths":["<path>"],"direction":"right","offset":1}' --json
        #     tt dev tzs set_spec_attr --form aapp320 --args '{"path":"<p>","kind":"field","attr":"can_edit","value":"Y"}'
        #     tt dev tzs list_open --json                                   # 无参数可省 --args
        # 动词全名：tt dev tzs --help     某个动词的参数与示例：tt dev tzs <动词> --help
  tt dev tzs formcheck --form42f <表单.42f> --tzs <包.tzs|表单.4fd> [--json]
        # **静态**比对：应然（.42f，编译期权威）↔ 实然（tzs 生成物里的 .4fd）
        # 不开引擎、不连服务器、不碰工作区 —— 只读上面两个文件
        # 差异分四组（存在性 / 编号 / 可写性 / 控件族），每条给 colName 与两侧值；
        #   拿不准的组合进"未判定"，**不计入错误**（契约 A15）
        # 退出码：0 一致（含只有未判定） / 2 参数或读取错 / 6 有确定差异
  tt dev tzs doctor [--json]         # 环境自检（引擎 / 设计器目录 / 工作区 / 管道名）
  tt dev tzs stop                    # 停本工作区的常驻引擎（不启动）
  tt dev tzs reap [--yes]            # 清理引擎重编后停不掉的孤儿守护进程

安装不在本命令组：skills 与 PATH 统一走 tt install skills / tt install path。

两条管线别用错：
  .tzc 代码包 → tzc export（渲染围栏工作区，改完 apply 写回；唯一写路径）
  .tzs 表单包 → tzs export 只解压（只读参考）；读写表单走 tzs 动词（设计器自己的引擎驱动）

工作区动词的 <dir> 可以省略：先 cd 进工作区，命令就不用再写目录。
  cd D:\pkg\capt110-ws
  tt dev tzc status          # = tt dev tzc status D:\pkg\capt110-ws
  tt dev tzc apply
  tt dev tzc rename adzi999_calc adzi999_count   # 省略 <dir> 时 rename 收 2 个位置参数

四个生命周期动词 + unlock 状态迁移 + selftest。
改框架区段必须先 unlock：解锁改变的是门，不是所有房间（锚点区段永远只读）。
  export  Package → Document → Workspace（含 git init），只读包
  status  报告工作区相对上次 export/apply 的改动（含行号，不写盘）
  verify  不写盘跑完整验证管线（gate1 + gate2）
  apply   验证管线全阶段 + 原子写包 + git commit（唯一会改 .tzc 的命令）

报错定位：验证失败会打印「prog.full.4gl:<行号>」+ 该行内容
（基线侧问题给「.tdev/base.full.4gl:<行号>」）；--json 里有 file/line/snippet 字段。

退出码：0 成功 / 2 包格式错 / 3 验证失败 / 4 写入被拒 / 5 IO·环境失败
tzs 线自己的码表是 0 / 1 / 2 / 4 / 5（没有 3）+ formcheck 的 6：
  1 引擎内部错 / 2 参数或环境错（含 manifest 拉不到）/ 4 设计器拒绝 / 5 传输或环境失败 / 6 有确定差异
`
