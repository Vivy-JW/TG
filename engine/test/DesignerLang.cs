using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.Reflection;
using System.Resources;
using System.Windows;
using System.Windows.Markup;

/// <summary>
/// 把设计器程序集里**内嵌的** `langs/zh-cn.xaml` 合进 app.Resources —— 探测程序共用的一份。
///
/// ## 为什么需要它
///
/// 在此之前，8 个探测程序各自硬编码了作者本机的设计器**反编译源码树**路径：
///
///     const string SRC = @"D:\我的项目\T100设计器";
///     ... File.OpenRead(Path.Combine(SRC, "SpecDesignerCommon", "langs", "zh-cn.xaml"))
///
/// `SRC` 是 `const`，任何 TZSCLI_* 变量都改不了它。于是**任何没有那棵树的机器上，
/// RoundTrip 一跑就 DirectoryNotFoundException** —— 而 `TestMiniCorpus*` 那三条回归
/// 正是靠 RoundTrip 的（engineExeAndDir 在**引擎不存在**时跳过，所以干净克隆上是绿跳，
/// 照着 BUILD.md 建完引擎才变红：**照文档做才会红**）。
///
/// 引擎本体早就把这条路改对了 —— 见 `src/Designer/Bootstrap.cs` 的 `MergeLanguages`，
/// 它的注释写得很明白："This used to read the one file out of the designer's decompiled
/// source tree … That reasoning was right about the path and stopped one step short: it is
/// not on a path, it is inside the assembly. `<Assembly>.g.resources` is a normal .NET
/// resource bundle, and ResourceReader hands back the entry as the ORIGINAL XAML (not baml)
/// … So the source tree is no longer a runtime dependency at all"。
///
/// 这份是那段逻辑在探测程序这一侧的同一实现。**没有直接调 Bootstrap 的原因**：探测程序是
/// `link` / `none` 模式编译的（见 build.sh 的 LINK_SRC），引用不到 `TzsCli.Designer` ——
/// 那是引擎本体的程序集。所以它只能放在 `engine/test/` 下，像 LIB 那样进每个程序的编译行。
///
/// **它是这一份，不是第 N 份**：8 个调用点共用它（build.sh 把本文件加进每种模式的 files）。
/// 新加探测程序时直接调 `DesignerLang.Merge(app, INSTALL)`，别再抄一份 ——
/// 仓库里有一条守卫盯着 `D:\我的项目` 这个字面量不许再出现。
/// </summary>
static class DesignerLang
{
    const string Lang = "zh-cn";

    /// <summary>把 <paramref name="install"/> 下每个 `SpecDesigner*.dll` 里内嵌的
    /// `langs/zh-cn.xaml` 合并进 <paramref name="app"/>.Resources。
    ///
    /// **两处都要**：实测 SpecDesignerCommon 带一份 88,737 字节的、SpecDesigner.Controls 另带
    /// 一份 450 字节的，缺一份就会让 FindResource("Message_...") 返回 null，并在很远的地方炸。
    /// 所以这里走目录、不加载单个文件。合并顺序按文件名排序，保证可复现。
    ///
    /// 一份都没合到时**抛异常**，不静默继续 —— 与 Bootstrap 同一条判据。</summary>
    public static void Merge(Application app, string install)
    {
        string[] files;
        try { files = Directory.GetFiles(install, "SpecDesigner*.dll"); }
        catch (Exception ex) {
            throw new Exception("读不到设计器目录 " + install + ": " + ex.Message);
        }
        Array.Sort(files, StringComparer.Ordinal);

        int merged = 0;
        foreach (string f in files) {
            Assembly asm;
            try { asm = Assembly.LoadFrom(f); } catch { continue; }   // 不是每个都能加载
            string bundle = asm.GetName().Name + ".g.resources";
            using (Stream s = asm.GetManifestResourceStream(bundle)) {
                if (s == null) continue;
                using (var rr = new ResourceReader(s)) {
                    foreach (DictionaryEntry e in rr) {
                        if (!string.Equals(e.Key as string, "langs/" + Lang + ".xaml",
                                           StringComparison.OrdinalIgnoreCase)) continue;
                        var stream = e.Value as Stream;
                        if (stream == null) continue;
                        app.Resources.MergedDictionaries.Add(
                            (ResourceDictionary)XamlReader.Load(stream));
                        merged++;
                    }
                }
            }
        }
        if (merged == 0)
            throw new Exception("设计器程序集里找不到 langs/" + Lang + ".xaml（install=" + install
                + "）；没有它 FindResource(\"Message_...\") 会返回 null 并在很远的地方炸");
    }
}
