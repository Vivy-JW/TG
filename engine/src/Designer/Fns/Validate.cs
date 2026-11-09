using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Reflection;
using System.Threading;
using Newtonsoft.Json.Linq;
using DS = TzsCli.Designer.Session;

namespace TzsCli.Designer.Fns
{
    /// <summary>
    /// `validate` and `set_excluded` (SPEC §11.24 (i), §11.22 校验/工具).
    ///
    /// `validate` is test/Probe.cs:411 RunValidate, hardened on all three points §11.24 (i)
    /// names -- and each of the three is a failure that looks like success, which is why the
    /// hardening is not optional polish:
    ///
    /// <list type="number">
    /// <item><b>SaveToForm() / SaveToTSD() first.</b> SpecificationInfo.FormElement and
    /// .TSDElement are ctor-time snapshots with private setters (:62 / :52, assigned once at
    /// :141), and the workers read exactly those. Skip the refresh and the validator inspects
    /// the file as loaded: every edit is invisible and the answer is always "clean".</item>
    ///
    /// <item><b>Not Thread.Sleep(600).</b> The P0 probe slept a fixed 600 ms after both workers
    /// went idle, which is a race that happened to pass. Both are BackgroundWorkers publishing
    /// DocumentErrorsEvent from their own thread, so the only sound signal is the event itself:
    /// a CountdownEvent signalled by the handler, a 60 s ceiling, and then a 300 ms window with
    /// no new event before the result is called final.</item>
    ///
    /// <item><b>ValidateForm restored to false in a finally, asserted afterwards.</b> It is not
    /// a UI preference; it is the first statement of XmlElement.CheckOverlapping, which runs on
    /// *every* SetAttribute -- including during the load of a different package (SPEC §11.23,
    /// §11.24 (i).1). A server that leaves it true starts throwing inside unrelated loads, and
    /// no single-package test can see it.</item>
    /// </list>
    ///
    /// The judgement is a multiset difference, never a boolean: newErrors = multiset(after) -
    /// multiset(before), split by severity. A clean run and a validator that threw on its
    /// BackgroundWorker and published nothing are indistinguishable from the outside
    /// (BackgroundWorker swallows DoWork exceptions), so the *report* is the evidence -- which
    /// is also why the acceptance test drives a deliberate duplicate name through it and
    /// requires exactly one new ERROR back.
    ///
    /// Two failures found while implementing this, both of which look exactly like "no errors
    /// found" from the outside, and both of which are the reason the positive control matters:
    ///
    /// * The designer re-subscribes the form validator's DoWork on every save, so the Nth
    ///   validate publishes every finding N times -- see <see cref="DropWorkers"/>.
    /// * Prism's one-argument Subscribe holds the handler *weakly*, so this module's own closure
    ///   can be collected while the subscription stays live -- see <see cref="SubscribeVia{T}"/>.
    /// </summary>
    public static class Validate
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["validate"]     = Run;
            into["set_excluded"] = SetExcluded;
        }

        // ------------------------------------------------------------------ baseline cache

        sealed class Rec {
            public string Type, Key, Description;
            public override string ToString() { return Type + " " + Key + " — " + Description; }
        }

        /// <summary>
        /// The pristine verdict per handle, captured lazily on that handle's first validate.
        ///
        /// Without it "new errors" would mean "relative to an empty file", and a file whose
        /// baseline is genuinely non-empty -- axmt500_wf(c).tzs reports 11 WARNINGs unmodified --
        /// would look broken on its first look and then silently change meaning on every later
        /// one. Keyed by handle, exactly like the session registry: ProgramKey is not unique
        /// (SPEC §11.24 (g)).
        /// </summary>
        static readonly Dictionary<string, List<Rec>> _baseline = new Dictionary<string, List<Rec>>(StringComparer.Ordinal);
        static readonly object _gate = new object();

        /// <summary>Drops a handle's cached baseline. Called by `close`: a reopened key is a
        /// genuinely new load, so carrying the old baseline across would compare two different
        /// files (SPEC §11.24 (g-1) item 3).</summary>
        public static void Forget(string handle) {
            lock (_gate) _baseline.Remove(handle);
        }

        /// <summary>Is there already a cached baseline for this handle?
        ///
        /// A composed verb that mutates and then reports a delta has to know this BEFORE it
        /// mutates: with no baseline, its first validate would run *after* the change and define
        /// the baseline as the broken state -- "new errors: none" by construction (§7.4).
        /// So it either seeds one first (one extra validate) or says so honestly.</summary>
        internal static bool HasBaseline(string handle) {
            lock (_gate) return _baseline.ContainsKey(handle);
        }

        // ------------------------------------------------------------------ validate

        internal static object Run(DS s0, JObject args) {
            DS s = TzsCli.Designer.Fns.Session.Resolve(s0, args);
            object si = s.Si;
            if (si == null) throw new TzsError("internal", "句柄没有 SpecificationInfo: " + s.Handle);
            RejectPath(args);

            object prefs = Prefs();
            var got = new List<Rec>();
            var arrived = new CountdownEvent(1);
            object evt = null, token = null;
            var total = Stopwatch.StartNew();   // the whole call, for elapsedMs
            var budget = new Stopwatch();       // the 60 s ceiling, started when the workers are
            string failure = null;              // started -- the ceiling is on waiting, not on
            try {                               // saving
                // The overlap pass is part of the designer's own validation, and it is the one
                // thing this switch turns on. Flipped before SaveToForm on purpose: overlap
                // detection is triggered by attribute writes, and SaveToForm writes plenty.
                if (prefs != null) Reflect.SetProp(prefs, "ValidateForm", true);

                evt = GlobalEvent("DocumentErrorsEvent");
                if (evt == null) throw new TzsError("internal", "取不到 DocumentErrorsEvent");
                token = Subscribe(evt, s, got, arrived);

                Reflect.Call(si, "SaveToForm");
                Reflect.Call(si, "SaveToTSD");
                DropWorkers(si);
                Reflect.Call(si, "InitTSDValidateWorker");
                Reflect.Call(si, "InitFormValidateWorker");

                object tsdv = Reflect.Prop(si, "_TSDValidater");
                object fmv  = Reflect.Prop(si, "_FormValidater");

                budget.Start();
                WaitIdle(tsdv, fmv, 60000, budget);
                WaitQuiet(arrived, 60000, budget);
            } catch (Exception ex) {
                failure = (ex.InnerException ?? ex).GetType().Name + ": " + (ex.InnerException ?? ex).Message;
            } finally {
                total.Stop();
                if (token != null && evt != null) { try { Unsubscribe(evt, token); } catch { } }
                // Point 3, and it is unconditional -- including on the throw path above.
                if (prefs != null) { try { Reflect.SetProp(prefs, "ValidateForm", false); } catch { } }
            }

            object restoredValue = prefs == null ? null : Reflect.Prop(prefs, "ValidateForm");
            if (restoredValue != null && (bool)restoredValue)
                throw new TzsError("internal",
                    "ValidateForm 没能恢复成 false：它是进程级的，留着会让别的包在加载时抛异常");

            List<Rec> after = Snapshot(got);
            List<Rec> before;
            lock (_gate) {
                if (!_baseline.TryGetValue(s.Handle, out before)) {
                    // First validate on this handle: this run *is* the baseline. Note what that
                    // buys and what it costs -- `after` is still reported in full, so a file
                    // that is dirty at first sight is visible in the response, it just does not
                    // show up as a *new* error. The acceptance test relies on exactly this
                    // ordering (baseline first, then the deliberate break).
                    before = after;
                    _baseline[s.Handle] = new List<Rec>(after);
                } else {
                    before = new List<Rec>(before);
                }
            }

            List<Rec> delta = Difference(after, before);
            var newErrors = new List<object>();
            var newWarnings = new List<object>();
            foreach (Rec r in delta) (r.Type == "ERROR" ? newErrors : newWarnings).Add(Rec2Obj(r));

            if (failure != null) throw new TzsError("internal", "校验失败: " + failure);

            return new Dictionary<string, object> {
                { "baseline",    List2Obj(before) },
                { "after",       List2Obj(after) },
                { "newErrors",   newErrors },
                { "newWarnings", newWarnings },
                { "elapsedMs",   (long)total.ElapsedMilliseconds },
            };
        }

        /// <summary>
        /// Manifest.cs used to declare an optional `path` on validate AND on verify ("只校验这条子树").
        /// Neither implements it: the designer's validators are handed one whole XElement
        /// (FormElement / TSDElement) and have no subtree entry point, and the fixed-point
        /// comparison is inherently whole-document. Failing loudly is the point -- a caller that
        /// asked for a subtree and silently received the whole form gets a superset it cannot
        /// distinguish from the answer it asked for.
        ///
        /// The declaration is GONE (2026-09-25, SPEC §11.9 item 13): the parameter could never do
        /// anything but come back E_NOT_IMPLEMENTED, whose kind is `internal` -- "report a bug, do
        /// not retry" -- for something the help was offering, and the CLI's "narrow it with `path`"
        /// hint led straight into it. Same call as `force` (item 4). This refusal stays as the
        /// backstop: a direct library caller or an older CLI still gets the honest reason instead of
        /// a silently whole-document answer.
        /// </summary>
        internal static void RejectPath(JObject args) {
            string p = TzsCli.Designer.Fns.Session.Str(args, "path");
            if (!string.IsNullOrEmpty(p))
                throw new TzsError("not_implemented",
                    "按子树校验尚未实现：设计器的校验器读的是整份 FormElement / TSDElement 快照，"
                    + "没有子树入口。不传 path 即为全表单校验，它的结论是子树结论的超集。");
        }

        /// <summary>
        /// Throws away the previous run's two validators so that the Init* calls below build
        /// fresh, singly-subscribed ones.
        ///
        /// This is not tidiness. InitFormValidateWorker does
        /// <c>this._FormValidater.DoWork += this.StartValidateForm;</c> on *every* call and never
        /// unsubscribes -- the one removal in StartValidateForm is `-= this.StartValidateTSD`, a
        /// no-op typo (StartValidateTSD is the handler StartValidateTSD itself removes). A
        /// BackgroundWorker's event is a multicast delegate, so the Nth validate on one
        /// SpecificationInfo runs the form validator N times and publishes every finding N times.
        /// Measured on aapp320: the deliberate duplicate produced 1 new ERROR on the first
        /// validate and 2 on the second, which would turn the T3 baseline (axmt500_wf reports 11
        /// WARNINGs unmodified) into 11 *fresh* warnings on the second call -- a phantom regression
        /// on a file nobody touched. The TSD worker is the well-behaved one and needs no help.
        ///
        /// Probe.cs never saw it because it compares counts and 2 > 0 reads the same as 1 > 0.
        ///
        /// Dropping the finished worker is safe: every use of _TSDValidater / _FormValidater in
        /// SpecificationInfo is null-guarded, and both are created on demand. The short wait is
        /// for the case where the previous call's 60 s ceiling expired first.
        /// </summary>
        static void DropWorkers(object si) {
            var pre = Stopwatch.StartNew();
            WaitIdle(Reflect.Prop(si, "_TSDValidater"), Reflect.Prop(si, "_FormValidater"), 5000, pre);
            Reflect.SetProp(si, "_TSDValidater", null);
            Reflect.SetProp(si, "_FormValidater", null);
        }

        /// <summary>
        /// Both workers must report not-busy before the quiet window can start. Polling is the
        /// only option: a BackgroundWorker's completion is marshalled through
        /// AsyncOperationManager, so IsBusy can stay true slightly after DoWork returns, and
        /// RunWorkerCompleted can post the last published events after that.
        /// </summary>
        static void WaitIdle(object tsdv, object fmv, long budgetMs, Stopwatch sw) {
            while (sw.ElapsedMilliseconds < budgetMs) {
                bool busy = IsBusy(tsdv) || IsBusy(fmv);
                if (!busy) return;
                Thread.Sleep(50);
            }
        }

        /// <summary>
        /// The quiet period: 300 ms with no new DocumentErrorsEvent. The CountdownEvent is
        /// re-armed after each signal rather than counted down once, because "has it produced
        /// anything yet" and "has it stopped producing" are different questions and only the
        /// second one decides when the answer is complete.
        /// </summary>
        static void WaitQuiet(CountdownEvent arrived, long budgetMs, Stopwatch sw) {
            while (sw.ElapsedMilliseconds < budgetMs) {
                if (arrived.Wait(300)) { arrived.Reset(); continue; }
                return;
            }
        }

        static bool IsBusy(object worker) {
            if (worker == null) return false;
            object b = Reflect.Prop(worker, "IsBusy");
            return b is bool && (bool)b;
        }

        /// <summary>
        /// multiset(after) - multiset(before), over (ErrorType, Key, Description).
        ///
        /// A multiset and not a set: the same (type, key, description) can legitimately occur
        /// twice, and a set difference would lose one of them. Order follows `after`, so the
        /// response lists new findings in the order the designer produced them.
        /// </summary>
        static List<Rec> Difference(List<Rec> after, List<Rec> before) {
            var have = new Dictionary<string, int>(StringComparer.Ordinal);
            foreach (Rec r in before) {
                string k = Sig(r);
                int c;
                have.TryGetValue(k, out c);
                have[k] = c + 1;
            }
            var outp = new List<Rec>();
            foreach (Rec r in after) {
                string k = Sig(r);
                int c;
                have.TryGetValue(k, out c);
                if (c > 0) have[k] = c - 1;
                else outp.Add(r);
            }
            return outp;
        }

        /// <summary>(char)1 rather than an empty join, or ("ERROR","ab","c") and
        /// ("ERROR","a","bc") would share one key: the fields are arbitrary designer text and
        /// contain no control characters. Spelled as an expression, not as a literal control
        /// byte in the source -- the same reason Json.cs writes its quotes as (char)34.</summary>
        const char SEP = (char)1;

        static string Sig(Rec r) { return r.Type + SEP + r.Key + SEP + r.Description; }

        static List<Rec> Snapshot(List<Rec> src) {
            lock (src) return new List<Rec>(src);
        }

        static List<object> List2Obj(List<Rec> src) {
            var list = new List<object>();
            foreach (Rec r in src) list.Add(Rec2Obj(r));
            return list;
        }

        static Dictionary<string, object> Rec2Obj(Rec r) {
            return new Dictionary<string, object> {
                { "type",        r.Type },
                { "key",         r.Key },
                { "description", r.Description },
            };
        }

        // ------------------------------------------------------------------ event plumbing

        /// <summary>
        /// EventAggregatorManager.Global.GetEvent&lt;DocumentErrorsEvent&gt;(). Every publish site
        /// in the designer goes through Global (SpecificationInfo.cs:374/419/437/508/566/587/623,
        /// SpecFieldNode.cs:60/72), so Global is the only place they can all be seen.
        /// </summary>
        static object GlobalEvent(string typeName) {
            object eam = Reflect.Prop2(Reflect.Find(Designer.A, "EventAggregatorManager"), "Global");
            if (eam == null) return null;
            Type evtType = Designer.A.GetType("SpecDesignerCommon.Events." + typeName);
            if (evtType == null) return null;
            foreach (var m in eam.GetType().GetMethods()) {
                if (m.Name != "GetEvent" || !m.IsGenericMethodDefinition || m.GetParameters().Length != 0) continue;
                return m.MakeGenericMethod(evtType).Invoke(eam, null);
            }
            return null;
        }

        /// <summary>
        /// Subscription is per call, not once per process. Global is process-wide and outlives
        /// every handle, so a handler installed at first validate and never removed would
        /// accumulate one entry per call -- and each of them would still be holding the list and
        /// the CountdownEvent of a call that finished long ago.
        /// </summary>
        static object Subscribe(object evt, DS s, List<Rec> into, CountdownEvent arrived) {
            Type payload = evt.GetType().BaseType.GetGenericArguments()[0];
            return typeof(Validate)
                .GetMethod("SubscribeVia", BindingFlags.Static | BindingFlags.NonPublic)
                .MakeGenericMethod(payload)
                .Invoke(null, new object[] { evt, s, into, arrived });
        }

        /// <summary>
        /// keepSubscriberReferenceAlive:true is NOT optional. Prism's one-argument
        /// Subscribe(Action&lt;T&gt;) defaults it to false, which stores the handler behind a
        /// WeakReference -- and this handler is a closure, so its display class is reachable only
        /// from that weak reference. Nothing else roots it, the GC is free to collect it at any
        /// point, and the subscription then goes quiet *without any error*: Publish succeeds,
        /// every other subscriber runs, and this one is simply gone.
        ///
        /// That is not theoretical. Measured on cpmp530: the identical call reported the duplicate
        /// name on some runs and reported nothing at all on others (0 new errors, `after` empty),
        /// while a second, raw subscription in the same process saw the ERROR arrive on every run.
        /// This is the same shape as §11.24 (i)'s warning about a validator that silently becomes
        /// a no-op: the only difference is that the failure lives in the subscriber rather than in
        /// the BackgroundWorker.
        ///
        /// Probe.cs has the same exposure (test/Probe.cs:400 subscribes with one argument and
        /// discards the token), which is part of why it needed the fixed Thread.Sleep(600) to look
        /// stable.
        /// </summary>
        static object SubscribeVia<T>(object evt, DS s, List<Rec> into, CountdownEvent arrived) {
            Action<T> h = delegate(T payload) { Collect(payload, s, into, arrived); };
            var keep = evt.GetType().GetMethod("Subscribe", new Type[] { typeof(Action<T>), typeof(bool) });
            if (keep != null) return keep.Invoke(evt, new object[] { h, true });
            var m = evt.GetType().GetMethod("Subscribe", new Type[] { typeof(Action<T>) });
            if (m == null) throw new Exception("没有 Subscribe(Action<T>) on " + evt.GetType());
            return m.Invoke(evt, new object[] { h });
        }

        static void Unsubscribe(object evt, object token) {
            foreach (var m in evt.GetType().GetMethods()) {
                if (m.Name != "Unsubscribe" || m.GetParameters().Length != 1) continue;
                m.Invoke(evt, new object[] { token });
                return;
            }
        }

        /// <summary>
        /// Filtered to this handle's ProgramKey. Global carries every package's errors, and while
        /// nothing here validates two packages at once, a long-lived server can have several
        /// handles open and the filter is what stops one handle's verdict from being reported as
        /// another's. A payload with no ProgramKey is accepted rather than dropped: dropping it
        /// would turn "the validator ran and found something" into "clean", which is the exact
        /// failure this whole file is designed to make impossible.
        /// </summary>
        static void Collect(object payload, DS s, List<Rec> into, CountdownEvent arrived) {
            object pk = Reflect.Prop(payload, "ProgramKey");
            if (pk != null && s.Key != null && !pk.Equals(s.Key)) return;

            var r = new Rec {
                Type        = Reflect.S(Reflect.Prop(payload, "ErrorType")),
                Key         = Reflect.S(Reflect.Prop(payload, "Key")),
                Description = Reflect.S(Reflect.Prop(payload, "Description")),
            };
            lock (into) into.Add(r);
            if (arrived.CurrentCount > 0) arrived.Signal();
        }

        /// <summary>
        /// PreferenceManager.Current.Settings -- the object CheckOverlapping dereferences. NOT
        /// TzpManager.Current (§11.24 (b)): PreferenceManager is a different singleton and is
        /// what the designer itself reads. Falls back to the private backing field because the
        /// public accessor is only guaranteed to exist, not to be the one Boot injected.
        /// </summary>
        static object Prefs() {
            object pm = Reflect.Prop2(Reflect.Find(Designer.A, "PreferenceManager"), "Current");
            if (pm == null) return null;
            object p = Reflect.Prop(pm, "Settings");
            if (p == null) p = Reflect.Prop(pm, "_preferenceModel");
            return p;
        }

        // ------------------------------------------------------------------ set_excluded

        /// <summary>
        /// `set_excluded handle path [excluded=true]` -> the node's new state.
        ///
        /// Runs the designer's own ExcludedUndoRedoCommand (SPEC §11.22 校验/工具 / `Excluded`),
        /// which is `SetExcluded(bool)` plus selection bookkeeping. Execute() calls SetExcluded
        /// *first*, so a failure in the UI-only tail (ComponentHelper.AddSelection) cannot lose
        /// the semantic change -- hence the catch, which rethrows only if the model really did
        /// not move.
        /// </summary>
        static object SetExcluded(DS s0, JObject args) {
            DS s = TzsCli.Designer.Fns.Session.Resolve(s0, args);
            TzsCli.Designer.Fns.Session.Touch(s);

            string path = TzsCli.Designer.Fns.Session.Str(args, "path");
            if (string.IsNullOrEmpty(path)) throw TzsError.Validation("set_excluded 需要一个 path");
            bool want = TzsCli.Designer.Fns.Session.Bool(args, "excluded", true);

            object el = s.FindByPath(path);
            if (el == null) throw TzsError.PathNotFound(path);
            string name = Reflect.S(Reflect.Prop(el, "Name"));
            if (string.IsNullOrEmpty(name))
                throw new TzsError("designer", "该元素没有控件代号，不能排除: " + path);

            object fsm = Reflect.Call(s.Si, "FindNodeByName", name);
            if (fsm == null) throw TzsError.NotFound("规格模型", name);

            Type ct = Reflect.Find(Designer.A, "ExcludedUndoRedoCommand");
            if (ct == null) throw new TzsError("internal", "找不到 ExcludedUndoRedoCommand");
            object cmd = Activator.CreateInstance(ct, new object[] { fsm, want });
            try {
                Reflect.Call(cmd, "Execute");
            } catch (Exception ex) {
                bool now = false;
                object isx = Reflect.Prop(fsm, "IsExcluded");
                if (isx is bool) now = (bool)isx;
                if (now != want)
                    throw new TzsError("designer", "排除设置失败: " + (ex.InnerException ?? ex).Message);
            }

            object after = Reflect.Prop(fsm, "IsExcluded");
            return new Dictionary<string, object> {
                { "path",     path },
                { "name",     name },
                { "excluded", after is bool && (bool)after },
                { "state",    TzsCli.Designer.Fns.Session.State(s) },
            };
        }
    }
}
