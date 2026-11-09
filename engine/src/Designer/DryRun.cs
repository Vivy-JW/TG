using System;
using System.IO;
using Newtonsoft.Json.Linq;
using DS = TzsCli.Designer.Session;

namespace TzsCli.Designer
{
    /// <summary>
    /// `dry_run` —— 跑一遍**真的**写动词，把它的答案整份留下，再让模型回到调用前。
    ///
    /// WHY THE ROLLBACK IS NOT "UNDO"  (2026-09-26, 实测推翻了原来的落法)
    ///
    /// 落法不是「应用 → 读 diff → 批量 undo，撤销栈是现成的」。**那条路是错的**，
    /// 而错法很具体：撤销栈管的是设计师的 command，而写动词的很大一部分写在 command 之外。
    /// 拿 15 个动词在 aapp320 上跑同一套判据（save → dry-run → save，比 sha256），结果分成两半：
    ///
    ///   · 只动布局的四个（set_layout_attr / insert_at / nudge / fit_size）逐字节回得去；
    ///   · 其余全部留残留 —— set_spec_attr 只多一个 status="u"（节点被摸过的标记），
    ///     rename_component / add_widget / set_local_string / set_excluded / convert_widget
    ///     各留各的，而 add_field / field_add 留下的是 UICreator 直接建进模型的那批规格节点
    ///     （.tsd 里多出 &lt;field name="apca_t.apcaent"&gt;、&lt;sfield name="lbl_apcaent"&gt;
    ///     和一条 TBinding，.bdx 跟着长）。UICreator 是工厂不是命令，它那半边从来没有进过撤销栈。
    ///
    /// 所以回滚不能建在撤销栈上，除非愿意接受"dry-run 有时改了你的表单还不说"。这里换成一条
    /// **构造上就精确**的路：
    ///
    ///   ① 先把**当前模型**渲染成一份临时包（就是 save 做的三件事：.4fd / .tsd / .bdx）；
    ///   ② 跑动词；
    ///   ③ 从句柄上把会话摘掉，再**从那份临时包重读回来**（句柄串不变）。
    ///
    /// 回滚因此是 reload —— 只是它读的不是盘上的源包（那会丢掉调用方**没保存**的改动），而是
    /// 调用前那一刻的渲染。这条路对每个动词都一样，不需要逐个动词证明，也不需要相信任何一半的
    /// 撤销栈。
    ///
    /// WHY THIS IS STILL HONEST ABOUT ITS COST
    ///
    /// 代价是真的，所以写在返回里，不藏着：会话被重建（loadMs 重新计、句柄回到 Loaded），模型
    /// 变成它**渲染出来**的那个样子（对每一份能通过语料定点判据的包，这与"没动过"是同一份字节
    /// —— 语料回归里那条 save→dry-run→save 必须逐字节相同的判据就是这句话的机械证明）。
    ///
    /// WHY A STATIC FLAG IS ACCEPTABLE HERE
    ///
    /// 传输是严格串行的（Rpc.cs 的头注释：设计器的活状态是进程级的，一次只能有一个请求在飞），
    /// 所以"当前这个请求是不是 dry-run"只能有一个答案，且没有第二个线程会读它。
    /// </summary>
    public static class DryRun
    {
        /// <summary>True while a dry-run body is executing. Read by the one place that writes to
        /// disk on a verb's behalf (Save.Run) -- 调用方自己要的 out 必须拦下来，而回滚用的那份
        /// 临时包是 Arm 自己写的，不走这里。</summary>
        public static bool Active;

        /// <summary>The marks for the request in flight. One at a time, like everything else here:
        /// the transport is strictly serial, so "the current dry run" is a single-valued thing.</summary>
        static Scope _scope;

        /// <summary>`dry_run` 在接受它的动词上被 Manifest.Check 定型成真布尔，所以这里读到的正常
        /// 就是一个 JTokenType.Boolean。走 Fns.Session.Bool 而不是 (bool)args[...]：它同时认
        /// "true"/"false" 字符串，与库内其它开关一个口径。</summary>
        public static bool Requested(JObject args) {
            return Fns.Session.Bool(args, "dry_run", false);
        }

        /// <summary>Before the body. Clears the previous request's marks, which is what makes a
        /// forgotten Exit harmless rather than a stale rollback on somebody else's model.</summary>
        public static void Enter() { _scope = null; Active = true; }

        /// <summary>Take the snapshot this request will be rolled back to.
        ///
        /// TWO CALLERS, and the second one is not optional. The dispatcher arms the ordinary case
        /// (it resolved the handle before the body). But a verb that resolves its OWN session --
        /// `field_add`, which finds it from `file` or `handle` inside the body -- reaches the
        /// dispatcher with nothing to arm, and the first cut of this code therefore rolled back
        /// nothing at all: the dry run ADDED the columns and reported `undoCommands: 0`. So the
        /// contract is "whoever first has the session arms it", and the guard makes it idempotent.
        ///
        /// Throws when the snapshot cannot be taken (a read-only directory, a full disk). That is
        /// the right moment to fail: the body has not run, so nothing has changed yet.</summary>
        public static void Arm(DS s) {
            if (!Active || _scope != null || s == null) return;
            _scope = Scope.Take(s);
        }

        /// <summary>After the body: hand the marks back and stop being a dry run. The caller rolls
        /// back with what it gets -- in the dispatcher's finally, because it has to happen when the
        /// body threw too.</summary>
        public static Scope Exit() {
            Active = false;
            Scope x = _scope;
            _scope = null;
            return x;
        }

        public sealed class Scope
        {
            DS _s;
            string _handle, _source, _scratch;
            string _sha;

            /// <summary>The bytes the snapshot was written from. Kept so the rollback does not
            /// depend on that file surviving -- see <see cref="Rollback"/>, which rewrites it if it
            /// has to. A few hundred KB in a daemon that is about to rebuild a whole package model
            /// anyway.</summary>
            byte[] _bytes;

            /// <summary>session-rebuild | none -- 见 Envelope。</summary>
            public string Mode = "session-rebuild";
            /// <summary>False only when the re-read failed; then Failure says why and Scratch is
            /// the file to open to get the caller's model back.</summary>
            public bool Complete = true;
            public string Failure;

            /// <summary>The scratch package's path, kept for the failure path (and for the
            /// undeletable-scratch case). Null after a clean rollback.</summary>
            public string Scratch { get { return _scratch; } }

            public double LoadMs, RenderMs;
            /// <summary>Whether <c>_existedAfterWrite</c> when the snapshot was taken. Reported
            /// only on the failure path, where it is the difference between "we never wrote it"
            /// and "something ate it".</summary>
            bool _existedAfterWrite;
            public string Sha { get { return _sha; } }

            Scope(DS s) { _s = s; _handle = s.Handle; _source = s.Path; }

            /// <summary>① Render the live model to a scratch package inside the source's own
            /// directory.
            ///
            /// Beside the source and not in %TEMP%: TzpManager refuses a package outside the
            /// workspace at LOAD time, so the scratch has to be openable again from the same
            /// process that wrote it. The source's own directory is inside the workspace by
            /// construction (the package was loaded from there).
            ///
            /// The name is `_tt_dry_` + pid + a counter, so a scratch left behind by a crash is
            /// recognisable and never mistaken for a corpus package (engine/make-manifest.sh and
            /// internal/dev/tzs's scratchPrefixes both exclude the prefix).</summary>
            internal static Scope Take(DS s) {
                var sc = new Scope(s);
                sc._scratch = ScratchPath(s.Path);
                var sw = System.Diagnostics.Stopwatch.StartNew();
                byte[] original = File.ReadAllBytes(s.Path);
                // write:true on purpose -- this IS a file write the dry run needs. It is the
                // snapshot, not the caller's `out`; DryRun.Active gates the latter, not this.
                byte[] rendered = Save.Run(original, sc._scratch, Fns.Session.Layout(s), s, true);
                sw.Stop();
                sc.RenderMs = sw.Elapsed.TotalMilliseconds;
                sc._sha = Save.Sha256Hex(rendered);
                // Kept so Rollback can rewrite the snapshot if the file goes missing, and checked
                // right here so that "the snapshot is gone" is diagnosed at the moment it is TRUE
                // rather than two statements before a reopen that fails for unexplained reasons.
                sc._bytes = rendered;
                sc._existedAfterWrite = File.Exists(sc._scratch);
                return sc;
            }

            static string ScratchPath(string sourcePath) {
                string dir = Path.GetDirectoryName(sourcePath);
                int pid = System.Diagnostics.Process.GetCurrentProcess().Id;
                for (int n = 0; n < 1000; n++) {
                    string p = Path.Combine(dir, "_tt_dry_" + pid + "_" + n + ".tzs");
                    if (!File.Exists(p)) return p;
                }
                throw new TzsError("internal", "临时包名用光了（" + dir + " 里有 1000 个 _tt_dry_*.tzs）");
            }

            /// <summary>The .4fd text of a package, for InstallLayout.</summary>
            static string FdText(string pkg) {
                byte[] zip = File.ReadAllBytes(pkg);
                string entry = Reflect.EntryName(zip, ".4fd");
                return entry == null ? null : Reflect.EntryText(zip, entry);
            }

            /// <summary>③ Put the session back on the snapshot. See the class header for why this
            /// and not Undo.
            ///
            /// Never throws: it runs in the dispatcher's finally, and a rollback that throws would
            /// replace the caller's real answer with a transport failure. What it could not do it
            /// records, and the failure path is designed to be recoverable -- the scratch package
            /// is left on disk precisely so that the caller can `open` it.
            ///
            /// THE SCRATCH IS RECONSTRUCTIBLE, AND THAT IS LOAD-BEARING (2026-09-26). A full-corpus
            /// run had ~14 of 65 dry runs come back with this file missing at reopen time -- written
            /// two statements earlier by this same process, and gone. It never reproduced when the
            /// dry-run guard ran alone (65/65 clean), and no deleter was ever identified; what
            /// matters more than the cause is that the promise must not rest on that file's
            /// survival. The bytes are already in hand, so a missing scratch is rewritten from them
            /// and the open is tried again -- and the reply says `recovered:true`, so the phenomenon
            /// stays visible instead of being papered over. A rollback that cannot even do that
            /// still leaves the caller a scratch to open (see Failure/Scratch).</summary>
            public void Rollback() {
                try {
                    _s.Close();
                    var sw = System.Diagnostics.Stopwatch.StartNew();
                    DS fresh = DS.Open(EnsureScratch(false), DS.TimeoutFromEnv());
                    sw.Stop();
                    LoadMs = sw.Elapsed.TotalMilliseconds;

                    // The session was loaded from the scratch, and the scratch is about to be
                    // deleted -- so the handle's own Path goes back to the package it is ABOUT.
                    // Leaving it at the scratch would break the next call that reads it (`save`
                    // reads File.ReadAllBytes(s.Path) as its repack base; `list_open` prints it),
                    // and the failure would be a FileNotFound long after the dry run. Measured
                    // 2026-09-26: that is exactly what the second dry run on one handle hit.
                    fresh.Path = _source;

                    // Same dance as `reload`: the freshly minted handle is burned, the caller's
                    // handle string is reused (Fns/Session.Reload has the long form of this).
                    string minted = fresh.Handle;
                    fresh.Handle = _handle;
                    // ...and the writer has to come from the SAME snapshot as the model, not from
                    // the handle's Path (which is the source file again). See InstallLayout.
                    Fns.Session.InstallLayout(fresh, FdText(_scratch));
                    Fns.Session.ReKey(minted, fresh);
                    // The validate baseline is NOT forgotten, and that is not an oversight: the
                    // model's content is the same as before this call (that is the whole point of
                    // rolling back to the snapshot), so the verdict captured earlier still refers
                    // to it. _loadedHash is likewise left alone -- the FILE on disk never moved.
                } catch (Exception ex) {
                    Complete = false;
                    Failure = Msg(ex) + "（临时包 " + (_scratch == null ? "已删" : (_existedAfterWrite ? "写完之后在" : "写完之后就不在"))
                            + "，此刻 File.Exists=" + (_scratch != null && File.Exists(_scratch)) + "）";
                }
                if (Complete) {
                    try { File.Delete(_scratch); _scratch = null; }
                    catch (Exception ex) { Failure = "临时包删不掉（不影响模型，但请手工删）: " + Msg(ex); }
                }
            }

            /// <summary>True once the snapshot has had to be written a second time because the
            /// first file was gone. Reported: it is a fact about this machine, not about the
            /// caller's request, and silently hiding it is how a flake becomes a mystery.</summary>
            public bool Recovered;

            /// <summary>The path to open, rewriting the snapshot first if the file has gone.</summary>
            string EnsureScratch(bool force) {
                if (!force && _scratch != null && File.Exists(_scratch)) return _scratch;
                if (_bytes == null) throw new TzsError("internal", "临时包丢了，而且没有留底字节");
                File.WriteAllBytes(_scratch, _bytes);
                if (!File.Exists(_scratch))
                    throw new TzsError("internal", "临时包写不进去: " + _scratch);
                Recovered = true;
                return _scratch;
            }
        }

        /// <summary>The dry-run envelope. One positive marker (`dryRun`), the verb's own answer
        /// under `preview`, and what the rollback did -- in that order of importance.
        ///
        /// `preview` is a key rather than a hoist of the verb's fields on purpose: a hoisted
        /// `noop:true` or `applied:true` would be the same marker in two contradictory positions at
        /// once ("it applied" here, "nothing landed" there), and that is the exact defect class the
        /// eval keeps finding. Inside `preview` every marker keeps the meaning §11.24 (a) gave it;
        /// `dryRun` is the marker for this fourth state.
        ///
        /// `scope` is null for a verb that accepted `dry_run` and has no model to put back --
        /// `save`, whose whole side effect is the file. Its `reverted.mode` is "none": nothing was
        /// undone because there was nothing to undo.</summary>
        public static JObject Envelope(object preview, string verb, string handle, Scope scope) {
            JToken pv = preview == null ? (JToken)JValue.CreateNull() : JToken.FromObject(preview);
            // A verb that resolves its OWN session (field_add) reaches the dispatcher with no handle
            // to name, while its answer carries one. Taking it from there is not decoration: the
            // caller's next question after a dry run is "which handle was that against".
            if (string.IsNullOrEmpty(handle)) {
                JObject pj = pv as JObject;
                JToken h = pj == null ? null : pj["handle"];
                if (h != null && h.Type == JTokenType.String) handle = (string)h;
            }

            var o = new JObject();
            o["dryRun"] = true;
            if (verb != null) o["verb"] = verb;
            if (handle != null) o["handle"] = handle;
            o["preview"] = pv;
            o["reverted"] = Reverted(scope);
            o["note"] = Note(scope);
            return o;
        }

        internal static JObject Reverted(Scope sc) {
            var rev = new JObject();
            if (sc == null) {
                rev["mode"] = "none";
                rev["complete"] = true;
                return rev;
            }
            rev["mode"] = sc.Mode;
            rev["complete"] = sc.Complete;
            rev["renderMs"] = Math.Round(sc.RenderMs, 1);
            if (sc.Recovered) rev["recovered"] = true;   // 快照文件丢过一次，用留底字节重写过
            if (sc.Complete) rev["loadMs"] = Math.Round(sc.LoadMs, 1);
            rev["sha256"] = sc.Sha;
            if (sc.Failure != null) {
                rev["error"] = sc.Failure;
                if (sc.Scratch != null) rev["scratch"] = sc.Scratch;
            }
            return rev;
        }

        static string Note(Scope sc) {
            const string tail = "要真做就去掉 dry_run 重发一次。";
            if (sc == null)
                return "dry-run：preview 是这次调用本该返回的结果（逐字）；这个动词不改模型，"
                     + "它唯一的副作用（写盘）已经被拦下。" + tail;
            if (!sc.Complete)
                return "dry-run：preview 是这次调用本该返回的结果（逐字）；模型**没能回滚**（" + sc.Failure
                     + "）。句柄已经关掉了" + (sc.Scratch == null ? ""
                        : "，你调用前的模型在 " + sc.Scratch + " —— open 它就是调用前的那一刻") + "。" + tail;
            return "dry-run：preview 是这次调用本该返回的结果（逐字）；模型已回到调用前"
                 + "（回滚方式是**重建会话**：先把模型渲染成一份临时包、跑一遍、再从那份临时包重读回来 —— "
                 + "撤销栈回不去这次写，实测见 SPEC §11.9 item 18）。副作用：句柄的 loadMs 重置、state 回到 Loaded；"
                 + "模型内容与调用前逐字节相同（判据是 save→dry-run→save 的 sha256）。" + tail;
        }

        /// <summary>The designer's own failure, unwrapped the way Rpc does it -- both
        /// Activator.CreateInstance and MethodInfo.Invoke wrap the cause in
        /// TargetInvocationException, whose message is always the same useless sentence.</summary>
        internal static string Msg(Exception ex) {
            Exception inner = ex;
            string s = inner.GetType().Name + ": " + inner.Message;
            while (inner.InnerException != null && inner.InnerException != inner) {
                inner = inner.InnerException;
                s += "  <<  " + inner.GetType().Name + ": " + inner.Message;
            }
            return s;
        }
    }
}
