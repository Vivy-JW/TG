using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.Xml.Linq;
using Newtonsoft.Json.Linq;
using TzsCli;

namespace TzsCli.Designer
{
    // Deliberately NOT TzsCli.Designer.Fns like the other modules in this directory. Fns/Session.cs
    // declares a static module class also called Session, and a type in the same namespace outranks
    // an enclosing one -- so inside .Fns the name `Session` would bind to that module and every
    // `Session s` parameter here would be CS0721 (static types cannot be used as parameters). Read.cs
    // and Attr.cs both chose the enclosing namespace for the same reason; Rpc.BuildMap dispatches by
    // class name, so the split costs nothing. Do not "tidy" it without renaming the module class.

    /// <summary>
    /// 页签（Folder 下的 Page）、Tab 顺序、以及两个工具函数。
    ///
    /// 这个文件是六个函数里最短的一个，但每一行都对应设计器里一处**已经存在**的行为——不是
    /// 重新实现的规则，而是把设计器自己的命令/服务按原样调起来：
    ///
    ///   add_page    AddComponetsUndoRedoCommand(ManagerForm.ExecuteAddPageBefore/After, :1021/:1030)
    ///   delete_page DeleteComponentsUndoRedoCommand（= ExecuteDelete 的那一条，:417）
    ///   set_tab_order / tab_action  ComponentTabIndexService（:1078-1170 的六个命令处理器）
    ///   base_data   SettingManager.Info_* 那几份 XML（MasterDetailView 的数据源）
    ///   set_code_template  SpecificationInfo.GetCodeTemplate/SetCodeTemplate
    ///
    /// 三件必须说清楚的事：
    ///
    /// 1. **模型改动和 .4fd 文本改动是两件事。** 命令只动模型；.4fd 是文本，由每 handle 的
    ///    FormWriter（Fns/Session.cs 的 Layout(s)，`save` 渲染的那一个）拼出来。两个都写，且
    ///    只写真正移动过的字节——这是 Edit.cs 的做法，也是 RoundTrip 不动点能成立的原因。
    /// 2. **tabIndex 是一张整表**（不是每个容器一份），所以 set_tab_order/tab_action 都是
    ///    "命令跑完 → 前后对比整张 tabIndex 表 → 把变了的写回文本"，而不是重算。
    /// 3. **Page 不是控件**：它没有规格节点、没有布局属性语义，只有名字。所以 add_page 的产物
    ///    是一个空壳 <Page name=...>，真正的约束是"父节点必须是 Folder"。
    /// </summary>
    public static class PageTab
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["add_page"]          = AddPage;
            into["delete_page"]       = DeletePage;
            into["set_tab_order"]     = SetTabOrder;
            into["tab_action"]        = TabAction;
            into["base_data"]         = BaseData;
            into["set_code_template"] = SetCodeTemplate;
        }

        // ================================================================ errors

        /// <summary>设计器自己的规则拒绝（SPEC §11.24 (a) 的 `designer`：不要重试，转述给用户）。
        ///
        /// code 用**小写**的 "designer" 而不是 "E_DESIGNER"：Rpc.Map 的表里 designer 是
        /// 一个 kind，`case "designer": wire="E_DESIGNER"; kind="designer"`；而 default 分支把任何
        /// 认不出的 E_* 码一律当 `internal` 发出去。发 E_DESIGNER 字面量得到的 kind 是 internal，
        /// 与 §11.24 (a) 的表相反（internal = 未预期的异常、上报）。线上的错误码两边都是 E_DESIGNER，
        /// 只有 kind 不同。Attr.cs 的 Refused 用的是 "E_DESIGNER"，那是它的文件，不在这里改。
        ///
        /// 载荷用 DetailedError 而不是私有子类，否则 Rpc 只能按普通 TzsError 处理而丢掉 detail。</summary>
        static DetailedError Refused(string message) { return new DetailedError("designer", message, new JObject()); }

        static DetailedError Refused(string message, JObject detail) {
            return new DetailedError("designer", message, detail ?? new JObject());
        }

        /// <summary>参数不合法但可以自纠：detail 带上 param 与合法值，SPEC §11.24 (a) 要求如此。
        /// 这些都不在 manifest 的静态校验覆盖范围内（manifest 只校验类型/枚举/必填），所以只能
        /// 由函数自己抛。</summary>
        static DetailedError BadParam(string param, string message, string[] legal) {
            var d = new JObject();
            if (param != null) d["param"] = param;
            if (legal != null) d["legal"] = Read.StrArray(legal);
            return new DetailedError("E_BAD_PARAM", message, d);
        }

        // ================================================================ plumbing

        /// <summary>SPEC §11.24 (b)：第一次写操作把 handle 从 Loaded 变成 Mutable。这里的命令
        /// （AddComponetsUndoRedoCommand / DeleteComponentsUndoRedoCommand）和 tabIndex 的
        /// setter（FormAttributesUndoRedoCommand）都会在 Execute 里按 key 去
        /// SettingManager.GetUndoRedoManager(key) 取管理器——没注册就抛 "No UndoRedoManager"，
        /// 或者更糟：静默什么都不做。所以注册必须在命令之前。</summary>
        static object Urm(Session s) {
            s.RegisterUndoRedo();
            return ((IDictionary)Reflect.Prop(Designer.SettingManager, "undoRedoManagerMap"))[s.Key];
        }

        /// <summary>AddThenExecute 而不是 Execute：命令的 Execute() 自己按 key 解析管理器，
        /// 那样管理器的 undo 栈就不记得这次改动（Edit.cs 也是 AddThenExecute）。</summary>
        static void Run(Session s, object cmd) {
            object urm = Urm(s);
            if (urm != null) { Reflect.Call(urm, "AddThenExecute", cmd); return; }
            Reflect.Call(cmd, "Execute");
        }

        static object NewCommand(string typeName, params object[] ctorArgs) {
            Type t = Reflect.Find(Designer.A, typeName);
            if (t == null) throw new TzsError("internal", "找不到命令类 " + typeName + "（设计器版本变了吗？）");
            return Activator.CreateInstance(t, ctorArgs);
        }

        /// <summary>每 handle 的 .4fd 文本写入器。契约见 Fns/Session.cs：写入函数必须通过这一个
        /// 实例改文本，因为 FormWriter 把改动存在自己内部——第二个实例不是同一份文档的第二个视图，
        /// 它就是第二份文档，`save` 永远看不到它。</summary>
        static FormWriter Layout(Session s) { return Fns.Session.Layout(s); }

        /// <summary>List&lt;XmlElement&gt; 用模型自己的类型构造。Activator 不能把
        /// List&lt;object&gt; 绑到 ctor 的 IEnumerable&lt;XmlElement&gt; 上，所以列表按元素的实际类型建
        /// （Attr.cs:432 同一个坑）。</summary>
        static IList ElList() {
            Type xt = Reflect.Find(Designer.A, "XmlElement");
            return (IList)Activator.CreateInstance(typeof(List<>).MakeGenericType(xt));
        }

        static string TypeOf(object el) { return Read.Str(Reflect.Prop(el, "Type")); }

        /// <summary>Look up a form element, or the caller's error. The same shape as Attr.cs's El,
        /// kept local so this file does not depend on another agent's private helpers.</summary>
        static object El(Session s, string path) {
            object el = s.FindByPath(path);
            if (el == null) throw TzsError.PathNotFound(path);
            return el;
        }

        // ================================================================ add_page

        /// <summary>
        /// 在 Folder 下新增一个页签。
        ///
        /// 设计器里"页签前/页签后"是同一个命令的两个 index（ManagedForm.ExecuteAddPageBefore
        /// :1021 用 xmlElement.Index，ExecuteAddPageAfter :1030 用 Index+1），被操作的是选中的
        /// 那个 Page，容器是它的 Parent。manifest 里 add_page 只有 `path`，没有 index，所以"前/后"
        /// 只能从 path 指到的东西读出来：
        ///
        ///   path = Folder  → 追加到末尾（index = 子节点数，命令内部会夹到 Nodes.Count）
        ///   path = Page    → 插在它后面（index = 它的 Index + 1），等价 ExecuteAddPageAfter
        ///
        /// 只接受这两种，而且容器必须是 Folder：Page 是 Folder 的专属子节点（manifest 的
        /// CONTAINERS 注释、TASKS 的 W3 说明都是这么写的），放进别的容器里设计器自己也没有入口。
        /// </summary>
        public static object AddPage(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string name = Read.Arg(a, "name");

            Urm(s);
            object el = El(s, path);
            string ptype = TypeOf(el);

            object container;
            int index;
            if (ptype == "Folder") {
                container = el;
                index = Session.ChildCount(el);                 // 末尾
            } else if (ptype == "Page") {
                container = Reflect.Prop(el, "Parent");
                if (container == null) throw Refused("页签 \"" + path + "\" 没有父节点，无法定位它的 Folder");
                index = (int)Reflect.Prop(el, "Index") + 1;     // 就是这个页签之后
            } else {
                throw Refused("add_page 的 path 必须是 Folder（新页签的父容器）或 Page（新页签插在它后面），"
                    + "收到的是 " + ptype + "：" + path);
            }
            if (TypeOf(container) != "Folder") {
                var d = new JObject();
                d["path"] = path;
                d["containerType"] = TypeOf(container);
                d["expected"] = "Folder";
                throw Refused("页签只能放进 Folder，而这个父容器是 " + TypeOf(container) + "："
                    + Read.Str(Reflect.Prop(container, "Name")), d);
            }

            Type cft = Designer.A.GetType("SpecDesignerCommon.Helpers.ComponentFactory");
            if (cft == null) throw new TzsError("internal", "找不到 ComponentFactory（设计器版本变了吗？）");
            object ctype = Enum.Parse(Reflect.Find(Designer.A, "ComponentType"), "Page", true);
            // 名字为空时 CreateEmptyComponent 自己走 GetNewName("Page", key)，也就是设计器
            // ExecuteAddPageBefore 传 "" 的后果——不是"无名页签"，是自动命名的 page_N。
            object page = Reflect.Call(cft, "CreateEmptyComponent", s.Key, ctype, name ?? "");
            string nm = Read.Str(Session.Raw(page, "name"));
            if ((bool)Reflect.Call(s.Si, "IsExists", nm))
                throw Refused("名称重复：\"" + nm + "\" 已经是这张表单上的另一个控件（SpecificationInfo.Add 会抛 Name is Exist）");

            IList one = ElList();
            one.Add(page);
            Run(s, NewCommand("AddComponetsUndoRedoCommand", one, container, index));

            // ---- .4fd：把新页签拼进文本。
            FormWriter w = Layout(s);
            var host = w.Index.ByPath(path);
            string containerPath = host == null ? null
                : (ptype == "Folder" ? host.Path : (host.Parent == null ? null : host.Parent.Path));
            if (containerPath == null || w.Index.ByPath(containerPath) == null)
                throw TzsError.NotFound("路径（.4fd 文本里）", path);
            // AddNode(parent, el, i) 的 i 是"插在 children[i] 之后"；-1 = 追加。
            int anchor = -1;
            if (ptype == "Page" && host.Parent != null) {
                for (int k = 0; k < host.Parent.Children.Count; k++)
                    if (ReferenceEquals(host.Parent.Children[k], host)) { anchor = k; break; }
            }
            w.AddNode(containerPath, (XElement)Reflect.Call(page, "ToXML"), anchor);

            var res = new JObject();
            res["path"] = containerPath + "/" + nm;
            res["name"] = nm;
            res["type"] = "Page";
            res["container"] = containerPath;
            res["index"] = index;
            res["layoutDelta"] = new JObject {
                { "parent", containerPath },
                { "added", Read.XAttrs((XElement)Reflect.Call(page, "ToXML")) }
            };
            return res;
        }

        // ================================================================ delete_page

        /// <summary>
        /// 删一个页签（连它的子节点）。
        ///
        /// 拒绝规则照抄 CanDeletePage（:1039）：元素**自己**必须是 Page，并且
        /// CanDelIncludeChildren（Form 自己、Form 的直接子节点、IsCantDel、以及任何后代不可删
        /// 都会让它为 false）。设计器在那种情况下只是把菜单项置灰，这里必须**说出来**而不是照样删——
        /// 一个被静默删掉的 Form 是没法回滚的。
        ///
        /// 命令用设计器自己的 DeleteComponentsUndoRedoCommand。它的构造函数有一个必须知道的副作用
        /// （DeleteComponentsUndoRedoCommand.cs:28）：当 list.Count == parent.Nodes.Count 且父节点是
        /// Folder/Table/Tree 时，它**把请求改成删父节点**（list 被就地改写）。所以命令建好之后要回读
        /// 那个 list，.4fd 才能删对东西——删错的话文本里会留下一个模型里已经不在的 Folder。
        /// </summary>
        public static object DeletePage(Session s, JObject a) {
            string path = Read.Need(a, "path");

            Urm(s);
            object el = El(s, path);
            string type = TypeOf(el);
            if (type != "Page") {
                var d = new JObject();
                d["path"] = path;
                d["type"] = type;
                d["expected"] = "Page";
                throw Refused("CanDeletePage 要求元素本身是 Page，收到的是 " + type + "：" + path
                    + "（删除普通控件是 delete，不是 delete_page）", d);
            }
            if (!(bool)Reflect.Prop(el, "CanDelIncludeChildren")) {
                var d = new JObject();
                d["path"] = path;
                d["canDelIncludeChildren"] = false;
                throw Refused("这个页签不能删：CanDelIncludeChildren 为 false"
                    + "（Form 自己、Form 的直接子节点、被标记 IsCantDel 的元素，或任何一个不可删的后代都会让它为 false）", d);
            }

            object parent = Reflect.Prop(el, "Parent");
            if (parent == null) throw Refused("页签 \"" + path + "\" 没有父节点");

            IList list = ElList();
            list.Add(el);
            // ExecuteDelete（ManagedForm:424）也对选中元素的绑定伙伴做同样的事；不这么做，
            // 删掉的控件在 .bdx 里还会留着一条绑定。
            object bind = Reflect.Prop(el, "BindElement");
            if (bind != null) { list.Add(bind); Reflect.Call(el, "ClearBinding"); }

            object cmd = NewCommand("DeleteComponentsUndoRedoCommand", parent, list);

            // 构造函数可能把这次删除改写成"删父节点"。回读它自己改写过的 list。
            bool escalated = list.Count == 1 && ReferenceEquals(list[0], parent);
            Run(s, cmd);

            FormWriter w = Layout(s);
            var host = w.Index.ByPath(path);
            string victim = path;
            if (escalated) {
                if (host == null || host.Parent == null)
                    throw TzsError.NotFound("路径（.4fd 文本里）", path);
                victim = host.Parent.Path;
            }
            w.RemoveNode(victim);

            var res = new JObject();
            res["path"] = path;
            res["name"] = Read.Str(Session.Raw(el, "name"));
            res["removed"] = victim;
            if (escalated) {
                res["escalated"] = true;
                res["note"] = "这是该父容器的最后一个子节点，设计器的命令改为删除父容器本身"
                    + "（DeleteComponentsUndoRedoCommand 构造函数的行为）；.4fd 删的也是它。";
            }
            res["layoutDelta"] = new JObject { { "removed", victim } };
            return res;
        }

        // ================================================================ Tab 顺序

        /// <summary>
        /// ComponentTabIndexService（SpecDesigner.FormEditor）。它按 PackageKey 缓存，所以 Get(key)
        /// 之后一定要 Register(FormNode) 把两个内部列表从当前模型重建。
        ///
        /// **Register 不会清空列表**——它调的是 1 参的 iteratorAllNodes(mainRoot)，那个重载只往
        /// _sourceList 里加；两个列表只在 0 参的 iteratorAllNodes()（只有 Show() 会调）和
        /// Dispose() 里被 Clear。设计器里这不是问题（Register 每次加载只跑一次），但服务是长驻的、
        /// 按 key 缓存的，所以同一个进程里第二次调 tab_action 会拿到上一次留下的列表：
        /// _sourceList 里是重复的元素，_tabIndexedList 里是上次"已编号前缀"，于是 ArrangeTabIndex
        /// 的编号从 9 开始（实测：两个 clear 连做，第二次是 8..15 而不是 1..8）。
        /// 所以这里先 Dispose() 清空两个列表，再 Register 重建——得到的就是新进程里
        /// Get+Register 的那个状态，也就是 Edit.exe `tab` 每次拿到的状态。
        /// </summary>
        static object TabService(Session s) {
            Type t = Reflect.Find(Designer.FE, "ComponentTabIndexService");
            if (t == null) throw new TzsError("internal", "找不到 ComponentTabIndexService"
                + "（SpecDesigner.FormEditor.dll 版本不匹配？）");
            object svc = Reflect.Call(t, "Get", s.Key);
            Reflect.Call(svc, "Dispose");                     // 两个列表清空（见上面那条）
            Reflect.Call(svc, "Register", Read.FormNode(s));
            return svc;
        }

        /// <summary>
        /// 命令跑完之后，把整张 tabIndex 表的前后差异写回 .4fd 文本，并返回动了多少处。
        /// 表是 Session.TabMap（整张表单，按 name-path 寻址），和 Edit.cs 用的是同一张。
        /// 没变的地方一个字节都不写——这是"只写移动过的字节"那条规则，也是 RoundTrip 不动点的来源。
        /// </summary>
        static int FlushTabIndex(Session s, FormWriter w, Dictionary<string, string> before) {
            Dictionary<string, string> after = s.TabMap(w);
            int moved = 0;
            foreach (var kv in before) {
                string now;
                if (!after.TryGetValue(kv.Key, out now) || now == kv.Value) continue;
                w.SetAttribute(kv.Key, "tabIndex", now);
                moved++;
            }
            return moved;
        }

        /// <summary>
        /// 重排 tabIndex。
        ///
        /// 不带 paths（paths 为空数组）＝按文档顺序重新编号 1..N，就是工具栏那个
        /// "Tab order specification" 按钮：ComponentTabIndexService.Show(true)，它重建
        /// _sourceList（文档顺序，且**包含** tabIndex 为空的元素）再由 SortSourceList 全部编号——
        /// 注意这跟 ArrangeTabIndex 不同，后者跳过空 tabIndex 的元素。这是 Edit.exe `tab`（无参）
        /// 的同一条路径，字节级可对比。
        ///
        /// 带 paths ＝ 这些元素排到最前，**按反序**逐个 SetAsFirst：每次调用都把当前的"已编号前缀"
        /// 冲到新头之后（ComponentTabIndexService.cs:81-88），所以倒着调才得到正着的顺序。
        /// 这是 Edit.exe `tab <path>...` 的同一条路径。
        /// </summary>
        public static object SetTabOrder(Session s, JObject a) {
            string[] paths = Read.ListArg(a, "paths");

            Urm(s);
            FormWriter w = Layout(s);
            var before = s.TabMap(w);
            object svc = TabService(s);

            var res = new JObject();
            var touched = new JArray();

            if (paths == null || paths.Length == 0) {
                Reflect.Call(svc, "Show", true);
                res["mode"] = "document-order";
                res["note"] = "按文档顺序重编号 1..N（= ComponentTabIndexService.Show(true)）；"
                    + "tabIndex 为空的元素也会被编号，这与 ArrangeTabIndex 不同";
            } else {
                var els = new List<object>();
                foreach (string p in paths) {
                    object el = El(s, p);
                    if (Session.Raw(el, "tabIndex") == null)
                        throw Refused("\"" + p + "\" 没有 tabIndex 属性——它不是参与 Tab 的控件类型"
                            + "（只有 mod-fd.spec 里列了 tabIndex 的那十几种才有）");
                    els.Add(el);
                }
                for (int i = els.Count - 1; i >= 0; i--) Reflect.Call(svc, "SetAsFirst", els[i]);
                foreach (string p in paths) touched.Add(p);
                res["mode"] = "first";
                res["requested"] = touched;
                res["note"] = "按反序逐个 SetAsFirst：每次调用都把已编号前缀冲到新头之后，"
                    + "所以倒着调才得到正着的顺序";
            }

            int moved = FlushTabIndex(s, w, before);
            res["changed"] = moved;
            res["tabIndex"] = TabTable(s, w);
            return res;
        }

        /// <summary>
        /// Tab 顺序动作六合一：设计器右键菜单 tabIndexControlContextMenu（Generic.xaml:856-880）的
        /// 六条命令，每条就是 ComponentTabIndexService 上的一个一行方法（ManagedForm:1078-1157）。
        ///
        /// manifest 的枚举名与设计器的菜单名不是一一对应的字面翻译，这里的对应关系是：
        ///
        ///   first → SetAsFirst        （Set as first index）
        ///   next  → SetAsNext         （Set as next index）
        ///   auto  → SetAsCurrent      （Set as current index：把文档顺序里到 m 为止的都编进去）
        ///   last  → ShiftCurrent      （Shift current index：m 接在已编号前缀之后，成为最后一个）
        ///   prev  → SwapSelected      （Swap with selected index：与前缀最后一个互换）
        ///   clear → SetAsNonTabable   （Set as non-tabable：tabIndex=""）
        ///
        /// first/next/clear 三个是唯一的（只有一条命令是这个意思）；auto/last/prev 三个是按语义
        /// 配的，manifest 没写下来，见交付报告里的说明。
        ///
        /// SwapSelected 在这里有个结构性限制：它取 `_tabIndexedList.Last()` 当作"当前索引"，
        /// 而 Register() 之后 _tabIndexedList 是空的（只有 _sourceList 被填），所以单独一次
        /// prev 会抛 InvalidOperationException（Sequence contains no elements）。设计器里没问题，
        /// 因为那个服务跨点击存活、前缀是一点点攒起来的；本接口每次调用都重开一次状态。
        /// 这里把它翻成一条明确的 E_BAD_PARAM，而不是让 LINQ 的异常冒出去。
        /// </summary>
        public static object TabAction(Session s, JObject a) {
            string action = Read.Need(a, "action");
            string[] paths = Read.ListArg(a, "paths");
            if (paths == null || paths.Length == 0)
                throw BadParam("paths", "tab_action 需要至少一个 path（动作作用在哪些元素上）", null);

            Urm(s);
            FormWriter w = Layout(s);
            var before = s.TabMap(w);
            object svc = TabService(s);

            var els = new List<object>();
            foreach (string p in paths) {
                object el = El(s, p);
                if (Session.Raw(el, "tabIndex") == null)
                    throw Refused("\"" + p + "\" 没有 tabIndex 属性——它不是参与 Tab 的控件类型");
                els.Add(el);
            }

            string method;
            switch (action) {
                case "first": method = "SetAsFirst"; break;
                case "next":  method = "SetAsNext"; break;
                case "clear": method = "SetAsNonTabable"; break;
                case "auto":  method = "SetAsCurrent"; break;
                case "last":  method = "ShiftCurrent"; break;
                case "prev":  method = "SwapSelected";
                    var probe = new JObject();
                    probe["action"] = action;
                    probe["legal"] = new JArray("first", "next", "last", "prev", "clear", "auto");
                    throw new DetailedError("E_BAD_PARAM",
                        "tab_action 的 prev 在这个接口里无法表达：SwapSelected 取 _tabIndexedList.Last() "
                        + "当作「当前索引」，而每次调用都会 Register() 一次，前缀列表始终是空的"
                        + "（设计器里它跨点击存活，前缀是一点点攒起来的）。要排到最前用 first，"
                        + "要排到最后用 last。", probe);
                default:
                    throw BadParam("action", "未知 action：" + action
                        + "；可用: first|next|last|prev|clear|auto", null);
            }

            foreach (object el in els) Reflect.Call(svc, method, el);

            int moved = FlushTabIndex(s, w, before);
            var res = new JObject();
            res["action"] = action;
            res["method"] = method;
            res["count"] = els.Count;
            res["changed"] = moved;
            res["tabIndex"] = TabTable(s, w);
            return res;
        }

        /// <summary>整张 tabIndex 表（name-path → 值），供调用方核对。</summary>
        static JObject TabTable(Session s, FormWriter w) {
            var o = new JObject();
            foreach (var kv in s.TabMap(w)) o[kv.Key] = kv.Value;
            return o;
        }

        // ================================================================ base_data

        sealed class Catalog {
            public string Key, File, Root, Row, Field, Label;
        }

        /// <summary>设计器自己那些选择器用的基础资料。每一条都是 SettingManager 上的一个 XML
        /// **字符串**（SettingManager.cs:587-640 从 <workspace>/mta/&lt;file&gt; 读进来），
        /// 而 MasterDetailView 就是拿这个字符串当 XmlDataProvider 的数据源、用 XPath "/*/*"
        /// （root 的子节点）当行、拿 id/desc 当两列（MasterDetailView.xaml.cs:134-153 与
        /// Filter() 的 id/desc 匹配）。所以这份目录能直接解析字符串给出，不需要那个 WPF 控件。
        ///
        /// 为什么这个函数重要：i_zoom 需要一个**合法的 zoom id**，chk_ref 需要一个合法的 check id，
        /// items 需要一个合法的数据项 id——它们全都只存在于这几份 XML 里，别处查不到。行里的
        /// param/rtn/opt 子元素也一并给出，因为那正是"填什么值"的答案。</summary>
        static readonly Catalog[] Cats = {
            new Catalog { Key = "items",       File = "items.xml",       Root = "items",    Row = "item",    Field = "Info_Items",       Label = "数据项" },
            new Catalog { Key = "zooms",       File = "zooms.xml",       Root = "zooms",    Row = "zoom",    Field = "Info_Zooms",       Label = "开窗设计" },
            new Catalog { Key = "checks",      File = "checks.xml",      Root = "checks",   Row = "check",   Field = "Info_Checks",      Label = "校验带值" },
            new Catalog { Key = "prog_rel",    File = "prog_rel.xml",    Root = "prog_rel", Row = "prog",    Field = "Info_ProgRel",     Label = "程序串查" },
            new Catalog { Key = "messages",    File = "messages.xml",    Root = "messages", Row = "message", Field = "Info_Messages",    Label = "提示讯息" },
            new Catalog { Key = "subroutines", File = "subroutines.xml", Root = "coms",     Row = "com",     Field = "Info_Subroutines", Label = "子程序" },
            new Catalog { Key = "libraries",   File = "libraries.xml",   Root = "coms",     Row = "com",     Field = "Info_Libraries",   Label = "函数库" },
        };

        /// <summary>一行最多带出多少个子元素。messages.xml 有 2.7 MB、zooms.xml 近 5 MB，
        /// 一行 JSON 的上限是 8 MB（Rpc.MaxLine），所以列表本身设上限并如实报告截断。
        /// manifest 没有 limit 参数，这个值是这里定的常数。</summary>
        const int MaxRows = 200;
        const int MaxChildren = 100;

        /// <summary>
        /// 基础资料目录。
        ///
        /// manifest 给的名字是 table / column（那一行本来是按 list_columns 抄的），在目录语境下
        /// 只能这样读：`table` = 查哪一本目录，`column` = 对 id 或 desc 做子串过滤
        /// （= MasterDetailView.Filter() 的行为，它不是精确匹配）。`table` 省略时返回目录清单本身，
        /// 那样调用方能先知道有哪几本、各有多少条。
        /// </summary>
        public static object BaseData(Session s, JObject a) {
            string want = Read.Arg(a, "table");
            string filter = Read.Arg(a, "column");

            if (string.IsNullOrEmpty(want)) {
                var list = new JArray();
                foreach (Catalog c in Cats) {
                    var o = new JObject();
                    o["table"] = c.Key;
                    o["label"] = c.Label;
                    o["file"] = "mta/" + c.File;
                    o["row"] = c.Row;
                    string src = Source(s, c);
                    o["loaded"] = src != null;
                    if (src != null) {
                        XElement root = XElement.Parse(src);
                        o["count"] = CountRows(root, c.Row);
                    }
                    list.Add(o);
                }
                var r0 = new JObject();
                r0["program"] = Read.Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
                r0["catalogs"] = list;
                r0["note"] = "没有给 table：返回目录清单。查一本要用 {\"table\":\"<上表之一>\","
                    + "\"column\":\"< id/desc 子串，可省略 >\"}。";
                return r0;
            }

            Catalog cat = Resolve(want);
            string source = Source(s, cat);
            if (source == null)
                throw TzsError.NotFound("基础资料 " + cat.Key,
                    "工作区里没有 mta/" + cat.File + "，SettingManager." + cat.Field + " 也是空的"
                    + "（LoadCommonData 只在文件存在时读它）");

            XElement xml;
            try { xml = XElement.Parse(source); }
            catch (Exception ex) {
                throw new TzsError("internal", "解析 " + cat.Field + " 失败（mta/" + cat.File + " 不是合法 XML）: " + ex.Message);
            }

            var rows = new JArray();
            int total = 0;
            foreach (XElement r in xml.Elements()) {
                if (r.Name.LocalName != cat.Row) continue;
                string id = Reflect.Attr(r, "id");
                string desc = Reflect.Attr(r, "desc");
                if (!Matches(filter, id, desc)) continue;
                total++;
                if (rows.Count >= MaxRows) continue;
                rows.Add(RowJson(r, id, desc));
            }

            var res = new JObject();
            res["program"] = Read.Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
            res["table"] = cat.Key;
            res["label"] = cat.Label;
            res["file"] = "mta/" + cat.File;
            res["row"] = cat.Row;
            if (filter != null) res["filter"] = filter;
            res["count"] = total;
            res["returned"] = rows.Count;
            res["truncated"] = rows.Count < total;
            if (rows.Count < total)
                res["note"] = "只回前 " + MaxRows + " 条（一行 JSON 有 8 MB 上限）；用 column 缩小范围。";
            res["rows"] = rows;
            return res;
        }

        /// <summary>一行：id/desc 两个属性（MasterDetailView 的两列）＋ 全部属性 ＋ 子元素。</summary>
        static JObject RowJson(XElement r, string id, string desc) {
            var o = new JObject();
            o["id"] = id;
            o["desc"] = desc;
            o["attrs"] = Read.XAttrs(r);
            var kids = new JArray();
            int n = 0;
            foreach (XElement c in r.Elements()) {
                n++;
                if (n > MaxChildren) continue;
                var co = new JObject();
                co["name"] = c.Name.LocalName;
                co["attrs"] = Read.XAttrs(c);
                kids.Add(co);
            }
            o["childCount"] = n;
            if (n > 0) o["children"] = kids;
            if (n > MaxChildren) o["childrenTruncated"] = true;
            return o;
        }

        static int CountRows(XElement xml, string row) {
            int n = 0;
            foreach (XElement e in xml.Elements()) if (e.Name.LocalName == row) n++;
            return n;
        }

        /// <summary>目录名解析。别名取行元素名 / 根元素名 / 文件名 / 中文名——都只在这几份 XML 里
        /// 出现过的东西，不是猜的；一个别名命中两本目录时明确报错而不是随便挑一本
        /// （"com" 同时是 subroutines 和 libraries 的行元素名）。</summary>
        static Catalog Resolve(string want) {
            var hits = new List<Catalog>();
            foreach (Catalog c in Cats) if (MatchKey(c, want)) hits.Add(c);
            if (hits.Count == 1) return hits[0];
            var keys = new string[Cats.Length];
            for (int i = 0; i < Cats.Length; i++) keys[i] = Cats[i].Key;
            if (hits.Count > 1) {
                var names = new List<string>();
                foreach (Catalog c in hits) names.Add(c.Key);
                throw BadParam("table", "table=\"" + want + "\" 同时命中 " + string.Join("/", names.ToArray())
                    + "，请用其中一个目录名", keys);
            }
            throw BadParam("table", "未知目录：" + want + "；可用: " + string.Join("|", keys), keys);
        }

        static bool MatchKey(Catalog c, string want) {
            return Same(c.Key, want) || Same(c.Row, want) || Same(c.Root, want)
                || Same(Path.GetFileNameWithoutExtension(c.File), want) || Same(c.File, want)
                || Same(c.Label, want);
        }

        static bool Same(string a, string b) {
            return a != null && string.Equals(a, b, StringComparison.OrdinalIgnoreCase);
        }

        /// <summary>MasterDetailView.Filter 的同一件事：id 或 desc 含子串（大小写不敏感）。</summary>
        static bool Matches(string filter, string id, string desc) {
            if (string.IsNullOrEmpty(filter)) return true;
            if (id != null && id.IndexOf(filter, StringComparison.OrdinalIgnoreCase) >= 0) return true;
            if (desc != null && desc.IndexOf(filter, StringComparison.OrdinalIgnoreCase) >= 0) return true;
            return false;
        }

        /// <summary>
        /// 一份基础资料的 XML 字符串。优先用 SettingManager 上那一份（就是 LoadCommonData 读进来的、
        /// 设计器自己那份，和选择器用的逐字节相同）；它为空但文件在的时候才去读文件——那说明
        /// specReferFiles 没列到它，而工作区里确实有。
        /// </summary>
        static string Source(Session s, Catalog c) {
            string txt = Read.Str(Reflect.Prop(Designer.SettingManager, c.Field));
            if (!string.IsNullOrEmpty(txt)) return txt;
            string ws = Read.Workspace(s);
            if (ws == null) return null;
            string f = Path.Combine(Path.Combine(ws, "mta"), c.File);
            if (!File.Exists(f)) return null;
            try { return File.ReadAllText(f); } catch { return null; }
        }

        // ================================================================ set_code_template

        /// <summary>
        /// 改 code_template（F/P/Q/W…，T100 的作业类型模板）。
        ///
        /// 合法值不是硬编码的四个字母：设计器自己的 CodeTemplateSelector（CodeTemplateSelector
        /// .xaml.cs:46-75）是从 `<workspace>/mta/code_template.xml` 的 &lt;kind id=…&gt; 里读的，
        /// 当前工作区那份有 F/P/Q/R/W 五个值（R = 报表作业）。所以这里也读那个文件，并把读到的
        /// id 当作唯一权威——一个不在文件里的 id 会是 E_BAD_PARAM，detail.legal 给出文件里的那串。
        ///
        /// 写的是 SpecificationInfo：它把值写在 TSDElement 的 &lt;other&gt;/&lt;code_template value=…&gt;
        /// 上并把 status 置 u；SaveToTSD 重建 TSDElement 时会把整个 &lt;other&gt; 原样带过去
        /// （SpecificationInfo.cs:1913 是克隆，只对 xelement3 那份删 other），所以 `save` 落盘的
        /// .tsd 里就是新值。.4fd 不参与，一个字都不动。
        ///
        /// manifest 里的 `path` **不是作用域**：code_template 是整张表单的属性（GetCodeTemplate /
        /// SetCodeTemplate 都只认 TSDElement.Element("other")），参数的 desc 却写着"name-path"。
        /// 这里把它当定位用——能解析到元素才继续，免得手滑传错 path 却"成功"改了全表；结果里如实
        /// 写明 scope=form。
        /// </summary>
        public static object SetCodeTemplate(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string tpl = Read.Need(a, "template");

            El(s, path);                       // 只为拒绝错路径；它不限定作用域

            string ws = Read.Workspace(s);
            if (ws == null) throw Refused("工作区路径读不到（SettingManager.CurrentSetting.Connection.Workspace 为空）");
            string file = Path.Combine(Path.Combine(ws, "mta"), "code_template.xml");
            if (!File.Exists(file))
                throw Refused("工作区里没有 mta/code_template.xml（设计器在这种情况下弹的是"
                    + " Message_CodeTemplateFileNotFound，没有可选值可给）：" + file);

            List<string> ids = new List<string>();
            var descs = new Dictionary<string, string>();
            XElement root;
            try { root = XElement.Load(file, LoadOptions.None); }
            catch (Exception ex) { throw new TzsError("internal", "解析 " + file + " 失败: " + ex.Message); }
            foreach (XElement k in root.Elements("kind")) {
                string id = Reflect.Attr(k, "id");
                if (string.IsNullOrEmpty(id)) continue;
                ids.Add(id);
                descs[id] = Reflect.Attr(k, "desc");
            }
            if (ids.Count == 0)
                throw Refused("mta/code_template.xml 里一个 <kind id=…> 都没有：" + file);

            string canonical = null;
            foreach (string id in ids) if (string.Equals(id, tpl, StringComparison.OrdinalIgnoreCase)) { canonical = id; break; }
            if (canonical == null)
                throw BadParam("template", "template=\"" + tpl + "\" 不在 " + file + " 的 <kind> 里；可用: "
                    + string.Join("|", ids.ToArray()), ids.ToArray());

            string old = Read.Str(Reflect.Call(s.Si, "GetCodeTemplate"));
            if (string.Equals(old, canonical, StringComparison.Ordinal)) {
                // SPEC §11.24 (a): a no-op is an OUTCOME, not a failure. This site has always returned
                // a success, but without the two markers the table requires (`noop:true` +
                // `code:E_NO_OP`) -- so a caller that forwards only the markers had no way to tell
                // "already this value" from a write this file declined to describe. Same shape as the
                // write path below, so the two answers read the same.
                var noop = new JObject();
                noop["path"] = path;
                noop["template"] = canonical;
                noop["old"] = old;
                noop["changed"] = false;
                noop["written"] = old;
                noop["desc"] = descs.ContainsKey(canonical) ? descs[canonical] : null;
                noop["scope"] = "form";
                noop["file"] = file;
                noop["legal"] = Read.StrArray(ids);
                noop["layoutDelta"] = JValue.CreateNull();   // .4fd 不动
                noop["noop"] = true;
                noop["code"] = "E_NO_OP";
                noop["note"] = "代码模板已经是 \"" + canonical + "\"，未修改任何值";
                return noop;
            }

            Urm(s);
            Reflect.Call(s.Si, "SetCodeTemplate", canonical);
            string now = Read.Str(Reflect.Call(s.Si, "GetCodeTemplate"));

            var res = new JObject();
            res["path"] = path;
            res["template"] = canonical;
            res["old"] = old;
            res["changed"] = now != old;
            res["written"] = now;
            res["desc"] = descs.ContainsKey(canonical) ? descs[canonical] : null;
            res["scope"] = "form";
            res["file"] = file;
            res["legal"] = Read.StrArray(ids);
            res["layoutDelta"] = JValue.CreateNull();   // .4fd 不动
            return res;
        }
    }
}
