using System;
using System.Collections.Generic;
using Newtonsoft.Json.Linq;

namespace TzsCli.Designer
{
    /// <summary>
    /// The operation log: one entry per **write request that carried an `op`**, written before the
    /// body runs and completed after it.
    ///
    /// WHY IT EXISTS (SPEC §11.9 item 20; asked for by the Go-side list, now in internal/dev/tzs/README.md)
    ///
    /// 从前只有一条禁令："请求一旦上线绝不重试"（协议里没有幂等键，重发一个 add_field 就是加两遍
    /// 列）。那条禁令治的是**重复**，没治**不知道**：调用方超时之后既不能重发，也问不出"到底写进去
    /// 没有"。这个日志就是那个问题唯一的机械答案 —— 调用方给自己这次写起一个名字（`op`），超时后
    /// 用 `list_ops` 问这个名字，拿到三态之一：
    ///
    ///   · <c>pending</c> —— 引擎**看见过**这次请求、还在做（或进程已经没了）。不许盲目重发。
    ///   · <c>ok</c> / <c>error</c> —— 做完了，附结果摘要 / 错误码。重发是安全的（要么它已经成了，
    ///     要么它明确失败了）。
    ///   · <b>没有这一条</b> —— 请求从来没到过引擎。重发是安全的。
    ///
    /// 为什么"没到过"这条答案是可信的：日志与守护进程同生共死，所以"空日志"只说明**这个进程**没
    /// 见过它。跨进程只能靠返回里的 <see cref="StartedAt"/> —— 调用方看到守护进程是刚才才起来的，
    /// 就该知道这份日志回答不了更早的问题。**不**把日志写到盘上：它的答案只在守护进程活着的这段时间
    /// 里有意义，而落盘会把"运维状态"塞进客户工作区或临时目录，两处都不该由引擎自己选。
    ///
    /// WHAT IT DELIBERATELY DOES NOT DO: 拒绝重复的 op。一个 op 名字用两次会记两条（都带 seq），
    /// 而"第二次要不要拦"是调用方的策略，不是引擎的 —— 拦下来就等于把 op 名字变成了一个全局命名
    /// 空间，而这个名字是调用方自己起的（两个不同的调用方用同一个 "step1" 是他们的自由，不是事故）。
    /// 引擎提供**事实**，不提供**策略**：seq / state / at 三样摆在那里，重没重发一眼可见。
    /// </summary>
    public static class OpLog
    {
        public sealed class Entry {
            /// <summary>Monotonic across the process. Reported so that a gap (entries evicted by
            /// the cap) is visible instead of silent.</summary>
            public long Seq;
            public string Op, Fn, Handle, Program;
            public string At;
            /// <summary>pending | ok | error</summary>
            public string State;
            public bool Dry;
            public double Ms;
            public string Code;
            public string Message;
            public JObject Summary;

            public JObject ToJson() {
                var o = new JObject();
                o["seq"] = Seq;
                o["op"] = Op;
                o["fn"] = Fn;
                if (Handle != null) o["handle"] = Handle;
                if (Program != null) o["program"] = Program;
                o["state"] = State;
                if (Dry) o["dryRun"] = true;
                o["at"] = At;
                if (State != "pending") o["ms"] = Math.Round(Ms, 1);
                if (Code != null) o["code"] = Code;
                if (Message != null) o["message"] = Message;
                if (Summary != null && Summary.Count > 0) o["summary"] = Summary;
                return o;
            }
        }

        /// <summary>Enough for a long session of careful work, small enough that a runaway caller
        /// cannot grow the daemon without bound. Evictions are counted and reported.</summary>
        const int MaxEntries = 500;

        static readonly List<Entry> _log = new List<Entry>();
        static readonly object _gate = new object();
        static long _seq;
        static long _dropped;

        /// <summary>When this process started serving. The one fact that lets a caller tell "this
        /// log is from the daemon that saw my request" from "this is a fresh daemon's empty log".</summary>
        public static readonly DateTime StartedAt = DateTime.Now;

        /// <summary>Record the attempt BEFORE the body runs. That order is the whole point: a
        /// request that kills the daemon mid-body must leave a `pending` entry, not nothing -- the
        /// difference between "it started" and "it never arrived" is the retry decision.</summary>
        public static Entry Begin(string op, string fn, string handle, string program, bool dry) {
            var e = new Entry {
                Op = op, Fn = fn, Handle = handle, Program = program, Dry = dry,
                State = "pending", At = Stamp(),
            };
            lock (_gate) {
                e.Seq = ++_seq;
                _log.Add(e);
                while (_log.Count > MaxEntries) { _log.RemoveAt(0); _dropped++; }
            }
            return e;
        }

        /// <summary>Complete the entry. `summary` is a few load-bearing fields pulled out of the
        /// verb's own result -- never the whole result (a form_tree answer does not belong in a log
        /// that is kept forever).</summary>
        public static void End(Entry e, string state, string code, string message, double ms, object result) {
            if (e == null) return;
            lock (_gate) {
                e.State = state;
                e.Code = code;
                e.Message = Clip(message, 200);
                e.Ms = ms;
                e.Summary = SummaryOf(result);
                if (string.IsNullOrEmpty(e.Handle)) {
                    JToken h = e.Summary == null ? null : e.Summary["handle"];
                    if (h != null && h.Type == JTokenType.String) e.Handle = (string)h;
                }
                // field_add 自报家门用的是 form（见它的返回体），与句柄上的 Program 是同一个东西。
                if (string.IsNullOrEmpty(e.Program)) {
                    JToken f = e.Summary == null ? null : e.Summary["form"];
                    if (f != null && f.Type == JTokenType.String) e.Program = (string)f;
                }
            }
        }

        /// <summary>Newest first. `op` is an exact match on the caller's own label -- that IS the
        /// question ("did MY op land"), and a substring match would answer it with somebody else's
        /// operation whenever one label is a prefix of another.</summary>
        public static JObject List(string op, int limit) {
            if (limit <= 0) limit = 20;
            List<Entry> hits = new List<Entry>();
            long total = 0, dropped = 0;
            lock (_gate) {
                dropped = _dropped;
                for (int i = _log.Count - 1; i >= 0; i--) {
                    Entry e = _log[i];
                    if (!string.IsNullOrEmpty(op) && e.Op != op) continue;
                    total++;
                    if (hits.Count < limit) hits.Add(e);
                }
            }
            var arr = new JArray();
            foreach (Entry e in hits) arr.Add(e.ToJson());
            var o = new JObject();
            o["daemonStartedAt"] = StartedAt.ToString("yyyy-MM-dd HH:mm:ss");
            o["count"] = total;                       // 匹配到几条（不只是回给你的这几条）
            o["returned"] = hits.Count;
            o["truncated"] = total > hits.Count;
            if (dropped > 0) o["dropped"] = dropped;  // 被容量挤掉的旧条目
            o["ops"] = arr;
            o["note"] = Note(op, total, dropped);
            return o;
        }

        static string Note(string op, long total, long dropped) {
            string what = string.IsNullOrEmpty(op) ? "所有写请求" : "op=" + op;
            string head = total == 0
                ? "没有 " + what + " 的记录 —— 这个守护进程没看见过它（请求从未到达，或者它是由另一个进程/另一次构建收到的；"
                  + "日志只记本进程，见 daemonStartedAt）"
                : what + "有 " + total + " 条记录";
            if (dropped > 0) head += "；另有 " + dropped + " 条旧记录已被容量挤掉（seq 会跳号）";
            return head + "。state: pending=还在做（别盲目重发）、ok/error=已经结束（重发安全）";
        }

        /// <summary>The handful of fields worth keeping. Chosen because they are the ones a caller
        /// branches on after a timeout: did it change anything (applied / clamped / noop / changed),
        /// what did it produce (sha256 / out), and for the composite verbs, the validation counts.
        /// `handle` is picked up too, for a verb (field_add) that resolves its own session.</summary>
        static readonly string[] KEYS = {
            "applied", "clamped", "noop", "changed", "code", "sha256", "out", "bytesOut",
            "newErrorCount", "newWarningCount", "handle", "form",
        };

        static JObject SummaryOf(object result) {
            var o = new JObject();
            JObject r = result as JObject;
            if (r == null && result != null) {
                try { r = JToken.FromObject(result) as JObject; } catch { return o; }
            }
            if (r == null) return o;
            foreach (string k in KEYS) {
                JToken v = r[k];
                if (v != null && v.Type != JTokenType.Null && v.Type != JTokenType.Object && v.Type != JTokenType.Array)
                    o[k] = v;
            }
            // field_add 把 save 的返回挂在 saved 下面，而"写进哪个文件、摘要是什么"正是超时后要问的。
            JObject saved = r["saved"] as JObject;
            if (saved != null) {
                foreach (string k in new[] { "out", "sha256", "written" }) {
                    JToken v = saved[k];
                    if (v != null && v.Type != JTokenType.Null) o["saved_" + k] = v;
                }
            }
            return o;
        }

        static string Stamp() { return DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss.fff"); }

        static string Clip(string s, int n) {
            if (s == null) return null;
            return s.Length <= n ? s : s.Substring(0, n) + "…";
        }
    }
}
