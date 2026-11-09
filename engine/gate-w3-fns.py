"""W3-fns gate -- drive all 33 MUTATING functions through one long-lived server.

WHY THIS EXISTS, AND WHAT IT IS NOT

gate-w3.py covers the shipping path (tzs-cli -> named-pipe daemon) and the READ surface over the
whole corpus, plus exactly one write (`nudge`). Of the 33 functions Manifest.cs declares
Mutating=true, 33 were never driven by any gate. Four batch drivers test the LEGACY path
(RoundTrip / AddField / Edit) and contain not one line of src/Designer/**. So "wave 3 delivered
29 functions" was, until this file, an unverified claim for 33 of them.

WHAT IT IS

One `tzs-server --stdio` process per corpus file, held open for the whole run. That is not a
convenience: it is a criterion of its own. SPEC / TASKS record that
ComponentTabIndexService.Register does not clear its two lists, so the SECOND tab operation in
one process continues from the first one's numbering (measured: a second `clear` numbers 8..15
instead of 1..8). A process-per-call harness cannot see that. PageTab.TabService calls Dispose()
before Register() to defend against it, and this gate is what would catch a regression.

Per file, in ONE process:

    open -> reads (form_tree / describe_kind x7 / list_* / get_component / find_component)
         -> validate(baseline)
         -> save S_pre
         -> [NEGATIVE probes: one per write group, bogus args]
         -> save S_mid          -- must be BYTE-IDENTICAL to S_pre (sha256)
         -> [every applicable write, in dependency order]
         -> validate(after)     -- newErrors must be 0
         -> save S_post         -- must DIFFER from S_pre, or every write was a silent no-op
         -> close

then, in a separate process, RoundTrip.exe on both the pristine file and S_post. The verdict is
BASELINE-RELATIVE, per SPEC 11.24 (d): adds/drops/pathAdds/pathDrops all 0, and `stale` equal to
the pristine file's own count -- NOT 0. cpmp530(c).tzs reports stale=2 before anything touches
it; gate-w3.py's first draft hardcoded 0 and failed it. Same trap, same rule.

THE FOUR VERDICTS, PER FUNCTION

    CHANGED   ok:true and a positive marker (applied / changed / added / deleted / removed)
    NOOP      ok:true and noop:true -- a declared SUCCESS (SPEC 11.24 (a): E_NO_OP is not a
              failure). Recorded, not failed.
    REFUSED   ok:false with a code the contract declares. Includes E_NO_OP-shaped refusals from
              the modules that still throw it (Semantic.NoOp).
    FAIL      anything else: an undeclared code, E_INTERNAL, E_NOT_IMPLEMENTED, a non-JSON line,
              an empty reply, a timeout, or a dead process.

SKIP is a first-class result too, and it must NAME the file precondition that is missing
("no Tree element", "no <tbl> row in the <table> section"). A bare skip is not evidence.

Usage: python gate-w3-fns.py [--files a,b,c] [--root DIR] [--only-fn f1,f2] [--keep] [--timeout S]

IS THIS STILL THE PLACE TO RUN IT -- no. It was reimplemented on the Go side and that is what runs
now, because this file cannot: it loads gate-w3.py from its own directory at import time and that
file is not in the repository. Keeping it anyway, on purpose: the header above is the original
argument for the four verdicts, for one-process-per-corpus-file, and for the baseline-relative
roundtrip rule -- the Go gate was written against it. What to run instead:

    TTZS_FNS=1 TTZS_CORPUS=%TEMP%\\ttws go test ./internal/dev/tzs -run TestFnsGate -timeout 30m -v
    (criteria + the two hard rules: internal/dev/tzs/fns_test.go, 根 TEST.md)

One thing did NOT carry over: the Go gate resolves the function list and the contract codes by
asking the engine (--manifest), where this file hardcodes both. Do not take its lists as current.
"""
import hashlib
import importlib.util
import json
import os
import queue
import subprocess
import sys
import threading
import time

HERE = os.path.dirname(os.path.abspath(__file__))

# --------------------------------------------------------------------------------- the shared bits
#
# gate-w3.py already owns the payload/frame plumbing, the RoundTrip parser and the baseline-relative
# verdict. Re-deriving any of it here would be a second copy that drifts, so it is imported and
# used as-is (_res / _roundtrip / _rt_parse / _rt_ok / _ws_of / _sha / _check / FROZEN_ERR).
# What gate-w3.py has no reason to own is an INTERACTIVE driver -- it never needed to read a result
# back to decide the next request. That is the one new piece here (§ Server).
_spec = importlib.util.spec_from_file_location('gate_w3', os.path.join(HERE, 'gate-w3.py'))
g3 = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(g3)

BIN = g3.BIN
ROOT = g3.ROOT
REPORT = os.environ.get('TZSCLI_FNS_REPORT') or os.path.join(
    os.environ.get('LOCALAPPDATA', '.'), 'Temp', 'gate-w3-fns-report.txt')
TMPDIR = os.path.join(os.environ.get('LOCALAPPDATA', '.'), 'Temp')

# A refusal is an ok:false carrying a code the contract declares (SPEC 11.24 (a)'s table plus the
# manifest's E_STD/E_SESS). E_NOT_IMPLEMENTED and E_INTERNAL are deliberately NOT here: "declared
# but no body" and "unexpected exception" are exactly the two things this gate exists to surface.
REFUSAL_CODES = {'E_BAD_PARAM', 'E_BAD_REQUEST', 'E_UNKNOWN_METHOD', 'E_NOT_FOUND', 'E_DESIGNER',
                 'E_KEY_IN_USE', 'E_NO_HANDLE', 'E_HANDLE_BUSY', 'E_PATH_NOT_FOUND',
                 'E_NO_SPEC_NODE', 'E_ATTR_NOT_WHITELIST'}

CONTAINERS = {'None', 'Grid', 'Group', 'ScrollGrid', 'Table', 'Tree', 'Page', 'HBox', 'VBox', 'Folder'}

# Steps after which the tree must be re-read: they add, remove or reparent elements, so every
# path derived from the previous snapshot may be stale by the time the next step runs.
DIRTY_FNS = {'add_widget', 'add_field', 'insert_at', 'delete', 'move', 'add_page', 'delete_page',
             'insert_semantic', 'convert_widget', 'convert_container', 'wrap', 'break_layout',
             'rename_component', 'add_action', 'delete_action'}

# The 33 the task names: Manifest.cs's Mutating=true set. `save` is NOT one of them (its Mutating
# is false on purpose -- SPEC 11.24 (b): "save 不改状态"), so it is driven as harness plumbing.
# `copy_component` is not one either any more: it was withdrawn from the product (it never
# copied -- see the report), so it now answers E_UNKNOWN_METHOD and must NOT be driven here.
WRITE_FNS = [
    'close', 'set_spec_attr', 'set_layout_attr', 'set_tree_source', 'rename_component',
    'add_widget', 'add_field', 'insert_at', 'delete', 'move', 'nudge', 'align', 'fit_size',
    'wrap', 'break_layout', 'convert_widget', 'convert_container',
    'add_page', 'delete_page', 'insert_semantic', 'add_action', 'delete_action', 'set_action_types',
    'set_local_string', 'set_items', 'set_progrel_programs', 'set_table_association',
    'set_spec_description', 'set_cited', 'set_tab_order', 'tab_action', 'set_excluded',
    'set_code_template',
]
# Which group each belongs to, so the aggregate can say "every group has a negative control".
GROUP_OF = {
    'close': '会话',
    'set_spec_attr': '属性', 'set_layout_attr': '属性', 'set_tree_source': '属性',
    'rename_component': '属性',
    'add_widget': '结构', 'add_field': '结构', 'insert_at': '结构', 'delete': '结构',
    'move': '结构', 'nudge': '结构', 'align': '结构', 'fit_size': '结构', 'wrap': '结构',
    'break_layout': '结构', 'convert_widget': '结构', 'convert_container': '结构',
    'add_page': '页签', 'delete_page': '页签',
    'insert_semantic': '语义/Action', 'add_action': '语义/Action', 'delete_action': '语义/Action',
    'set_action_types': '语义/Action',
    'set_local_string': '多语言/选项/串查', 'set_items': '多语言/选项/串查',
    'set_progrel_programs': '多语言/选项/串查', 'set_table_association': '多语言/选项/串查',
    'set_spec_description': '多语言/选项/串查', 'set_cited': '多语言/选项/串查',
    'set_tab_order': 'Tab 顺序', 'tab_action': 'Tab 顺序',
    'set_excluded': '校验/工具', 'set_code_template': '校验/工具',
}

report_lines = []
outcomes = {}          # (file, fn) -> list of one-line verdict strings
skips = {}             # (file, fn) -> reason
neg_results = []       # (file, group, fn, code, refused_ok)


def say(s=''):
    try:
        print(s)
    except Exception:
        print(s.encode('ascii', 'replace').decode('ascii'))
    report_lines.append(s)


# ================================================================================= the transport

class Server:
    """A live `tzs-server --stdio` process, driven request by request.

    gate-w3.py's `srv()` hands the whole batch to subprocess.run and parses everything at the end --
    the right shape for read-only sweeps, and useless for a sequence in which the arguments of step
    N+1 come out of step N's result (`wrap` mints a container whose name nobody can predict, then
    `break_layout` has to be pointed at it). Same wire, same frames: the reply is handed to
    gate-w3.py's own `res()` so the two drivers cannot classify a frame differently.

    The reader runs on a thread and feeds a Queue so that a hung request is a TIMEOUT verdict
    rather than a hung gate. stderr goes to a file: the server writes bootstrap diagnostics there,
    and leaving it as an unread pipe would fill and block the child.
    """

    def __init__(self, ws, tag):
        self.tag = tag
        self.errpath = os.path.join(TMPDIR, 'w3fns_%s.err' % tag)
        self._err = open(self.errpath, 'wb')
        self.p = subprocess.Popen(
            [os.path.join(BIN, 'tzs-server.exe'), '--stdio'],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=self._err,
            env=dict(os.environ, TZSCLI_WS=ws), cwd=BIN)
        self.q = queue.Queue()
        self.n = 0
        self.dead = False
        self._t = threading.Thread(target=self._read, name='w3fns-reader')
        self._t.daemon = True
        self._t.start()

    def _read(self):
        try:
            for line in iter(self.p.stdout.readline, b''):
                self.q.put(line)
        except Exception:
            pass
        self.q.put(None)          # EOF

    def call(self, fn, args, timeout=300):
        """One request, one reply. Never raises: a dead or silent server is a verdict."""
        self.n += 1
        i = self.n
        if self.dead:
            return i, False, {}, {'code': 'E_HARNESS_DEAD', 'kind': 'internal',
                                  'message': '进程已退出，不再发送'}
        try:
            self.p.stdin.write((json.dumps({'id': i, 'fn': fn, 'args': args},
                                           ensure_ascii=False) + '\n').encode('utf-8'))
            self.p.stdin.flush()
        except Exception as e:
            self.dead = True
            return i, False, {}, {'code': 'E_HARNESS_STDIN', 'kind': 'internal', 'message': str(e)}
        try:
            raw = self.q.get(timeout=timeout)
        except queue.Empty:
            self.dead = True
            return i, False, {}, {'code': 'E_HARNESS_TIMEOUT', 'kind': 'internal',
                                  'message': '%s 在 %ss 内没有回应' % (fn, timeout)}
        if raw is None:
            self.dead = True
            return i, False, {}, {'code': 'E_HARNESS_EOF', 'kind': 'internal',
                                  'message': '服务器没有回帧就退出了'}
        txt = raw.decode('utf-8', 'replace').strip()
        if not txt.startswith('{'):
            return i, False, {}, {'code': 'E_HARNESS_PARSE', 'kind': 'internal',
                                  'message': '非 JSON 帧: ' + txt[:160]}
        try:
            frame = json.loads(txt)
        except ValueError as e:
            return i, False, {}, {'code': 'E_HARNESS_PARSE', 'kind': 'internal',
                                  'message': '帧解析失败: %s; %s' % (e, txt[:160])}
        res, err, ok = g3.res({i: frame}, i)      # gate-w3.py's own classifier: (result, error, ok)
        return i, ok, res, err
    def close(self):
        try:
            self.p.stdin.close()
        except Exception:
            pass
        try:
            self.p.wait(timeout=20)
        except Exception:
            try:
                self.p.kill()
                self.p.wait(timeout=10)
            except Exception:
                pass
        try:
            self._err.close()
        except Exception:
            pass


# ================================================================================= classification

def verdict(ok, res, err):
    if ok:
        if res.get('noop') is True:
            return 'NOOP', 'noop'
        if res.get('closed') is False:
            return 'NOOP', 'already-closed'
        if res.get('changed') is False or res.get('code') == 'E_NO_OP':
            return 'NOOP', res.get('code') or 'changed:false'
        mark = [k for k in ('applied', 'clamped', 'added', 'deleted', 'removed', 'created',
                            'reused', 'escalated', 'standalone') if res.get(k) is True]
        if res.get('changed') is True or mark:
            return 'CHANGED', ','.join(mark) or 'changed:true'
        if isinstance(res.get('delta'), list) and res['delta']:
            return 'CHANGED', 'delta'
        if res.get('code') == 'E_NO_OP':
            return 'NOOP', 'E_NO_OP (success)'
        return 'CHANGED', 'ok'
    code = err.get('code') or '?'
    if code in REFUSAL_CODES:
        return 'REFUSED', code
    return 'FAIL', code


def short(o, n=150):
    try:
        s = json.dumps(o, ensure_ascii=False)
    except Exception:
        s = str(o)
    return s if len(s) <= n else s[:n] + '...'


def widget_columns(ctx, maxtables=8):
    """(table, column) pairs whose col_attr carries a non-empty widget -- via the DECLARED tools.

    `add_field`'s blocking precondition is a column with a `col_attr@widget`, and list_tables +
    list_columns are exactly the two functions that answer it (list_columns returns each column's
    `colAttr` verbatim). Until this rebuild BOTH of them NREd -- their manifest entries said
    NeedsHandle=false while the bodies dereferenced the session -- so this used to read
    mta/tables.xml and <module>/tbl/<table>.tbl by hand. That workaround is gone with the defect;
    the declared surface is used instead.

    Cached per file: the pair list does not change while the gate runs, and each call parses a .tbl.
    """
    if ctx.colcols is not None:
        return ctx.colcols
    out = []
    _i, ok, res, err = ctx.srv.call('list_tables', {})
    if ok:
        for t in (res.get('tables') or [])[:maxtables]:
            tn = t.get('name')
            if not tn:
                continue
            _i, ok2, res2, err2 = ctx.srv.call('list_columns', {'table': tn})
            if not ok2:
                continue
            for col in (res2.get('columns') or []):
                ca = col.get('colAttr') or {}
                if (ca.get('widget') or '').strip():
                    out.append((tn, col.get('name')))
    ctx.colcols = out
    return out


def tsd_table_info(path):
    """(tbl names, sr names) out of the package's own .tsd, without a session.

    `set_table_association`'s flag branch needs a live <sr name=thisElement>, and the element it
    names is the only path that branch accepts. Reading the two name lists here means the step can
    point at one instead of provoking the refusal on purpose.
    """
    import re
    import zipfile
    try:
        z = zipfile.ZipFile(path)
        nm = [n for n in z.namelist() if n.lower().endswith('.tsd')]
        if not nm:
            return [], []
        t = z.read(nm[0]).decode('utf-8', 'replace')
        i = t.find('<table')
        j = t.find('</table>', i) if i >= 0 else -1
        seg = t[i:j] if i >= 0 and j > i else ''
        return (re.findall(r'<tbl\s+name="([^"]+)"', seg),
                re.findall(r'<sr\s+name="([^"]+)"', seg))
    except Exception:
        return [], []


def note_of(ok, res, err):
    return short(res if ok else err)


# ================================================================================= tree helpers

def build_nodes(tree_res):
    root = tree_res.get('root') or {}
    out = []
    if not root:
        return out

    def walk(n, parent):
        out.append({'path': n.get('path'), 'tag': n.get('tag'), 'name': n.get('name'),
                    'parent': parent})
        for c in (n.get('children') or []):
            walk(c, n.get('path'))
    walk(root, None)
    return out


class Ctx(object):
    """Per-file state: the live server, the current tree, and every result so far."""

    def __init__(self, srv, target, ws, idx):
        self.srv = srv
        self.target = target
        self.ws = ws
        self.idx = idx
        self.h1 = None
        self.h2 = None
        self.N = []
        self.tree = {}
        self.kinds = {}
        self.spec_nodes = []
        self.strings = 0
        self.dirty = True
        self.colcols = None      # (table, column) pairs with a col_attr widget, cached
        self.res = {}          # step id -> result payload
        self.log = []

    def ensure(self):
        if not self.dirty:
            return
        _i, ok, res, err = self.srv.call('form_tree', {'handle': self.h1, 'depth': 99})
        if not ok:
            raise RuntimeError('form_tree 失败: ' + short(err))
        self.tree = res
        self.N = build_nodes(res)
        self.dirty = False

    # ---- node lookups (all recomputed against the CURRENT tree) --------------------------------

    def node(self, path):
        for n in self.N:
            if n['path'] == path:
                return n
        return None

    def leaves(self):
        return [n for n in self.N if n['tag'] not in CONTAINERS and n['tag'] != 'Form']

    def of_tag(self, *tags):
        want = set(tags)
        return [n for n in self.N if n['tag'] in want]

    def named(self, name):
        for n in self.N:
            if n['name'] == name:
                return n
        return None

    def parent(self, node):
        return self.node(node['parent']) if node and node.get('parent') else None

    def spec_names(self, kind):
        return [x['name'] for x in self.spec_nodes if x.get('kind') == kind]

    def element_for_spec(self, kind):
        """A spec node whose Name is also a live element name -- what FindNodeByName requires."""
        names = set(n['name'] for n in self.N)
        for nm in self.spec_names(kind):
            if nm in names:
                return self.named(nm)
        return None

    def get(self, path, **kw):
        """(ok, result, error) for one get_component -- the id carries no information here."""
        args = {'handle': self.h1, 'path': path}
        args.update(kw)
        _i, ok, res, err = self.srv.call('get_component', args)
        return ok, res, err

    def first_leaf_named(self, name):
        n = self.named(name)
        if n and n['tag'] not in CONTAINERS:
            return n
        return None


# ================================================================================= the write plan

def plan(ctx):
    """The ordered list of write steps. Each entry is a dict:

        id     unique label (several sub-invocations share one id)
        fn     the manifest function name
        build  callable(ctx) -> args dict | ('SKIP', reason) | list of those
        dirty  the step moves other elements' name-paths (wrap / break / rename / copy)

    `build` runs at execution time against the CURRENT tree, so a step never carries a stale
    path past a structural edit.
    """
    P = []

    def step(id_, fn, build, dirty=False, structural=False, candidates=False):
        P.append({'id': id_, 'fn': fn, 'build': build, 'dirty': dirty,
                  'structural': structural, 'candidates': candidates})

    # ---- a stable set of leaves to operate on ------------------------------------------------
    def pick_leaf(k=0):
        ctx.ensure()
        ls = ctx.leaves()
        return ls[k] if len(ls) > k else None

    def leaf_with_spec(kind='field'):
        ctx.ensure()
        cand = ctx.element_for_spec(kind)
        if cand:
            return cand
        for n in ctx.leaves():
            return n
        return None

    # ---- 属性 ----------------------------------------------------------------------------------

    def b_set_layout_batch(c):
        c.ensure()
        ls = c.leaves()
        if len(ls) < 2:
            return ('SKIP', '文件里不足 2 个非容器元素（批量 set_layout_attr 需要至少 1 个可改元素）')
        cur = c.get(ls[0]['path'])
        attr, val = 'gridWidth', None
        if cur[0]:
            g = (cur[1].get('layout') or {}).get('gridWidth')
            if g is not None and g.isdigit():
                val = str(int(g) + 1)
        if val is None:
            attr, val = 'posX', '3'
        # `path` is required by the manifest even on the batch form -- Struct.PathsArg's own
        # comment: "delete declares BOTH ... so Manifest.Check refuses a batch call that omits
        # `path` before this body ever runs". Passing it is what makes `paths` take over.
        return {'path': ls[0]['path'], 'paths': [ls[0]['path'], ls[1]['path']],
                'attr': attr, 'value': val}

    def b_set_layout_one(c):
        c.ensure()
        n = pick_leaf(0)
        if not n:
            return ('SKIP', '文件里没有非容器元素')
        ok, res, _e =c.get(n['path'])
        if not ok or 'gridWidth' not in (res.get('layout') or {}):
            return ('SKIP', '元素 %s 的布局里没有 gridWidth' % n['name'])
        g = res['layout']['gridWidth']
        if not (g or '').isdigit():
            return ('SKIP', 'gridWidth 不是整数: %r' % g)
        return {'path': n['path'], 'attr': 'gridWidth', 'value': str(int(g) + 1)}

    def b_set_spec(c, attr):
        def build(cc):
            cc.ensure()
            n = leaf_with_spec()
            if not n:
                return ('SKIP', '没有带规格节点的元素')
            ok, res, _e =cc.get(n['path'])
            if not ok:
                return ('SKIP', 'get_component 失败: ' + short(res))
            f = ((res.get('spec') or {}).get('field') or {}).get('attrs') or {}
            if attr not in f:
                return ('SKIP', '元素 %s 的 field 节点没有属性 %s' % (n['name'], attr))
            old = f[attr]
            new = 'Y' if old != 'Y' else 'N'
            return {'path': n['path'], 'kind': 'field', 'attr': attr, 'value': new}
        return build

    step('set_layout_attr:batch', 'set_layout_attr', b_set_layout_batch)
    step('set_layout_attr:one', 'set_layout_attr', b_set_layout_one)
    step('set_spec_attr:can_query', 'set_spec_attr', b_set_spec(None, 'can_query'))
    step('set_spec_attr:req', 'set_spec_attr', b_set_spec(None, 'req'))

    def b_spec_desc(c):
        c.ensure()
        n = leaf_with_spec()
        if not n:
            return ('SKIP', '没有带规格节点的元素')
        return {'kind': 'field', 'path': n['path'], 'content': 'cli gate-w3-fns probe description'}
    step('set_spec_description', 'set_spec_description', b_spec_desc)

    # ---- 多语言 / 选项 --------------------------------------------------------------------------

    def b_local_string(c, text):
        def build(cc):
            cc.ensure()
            n = pick_leaf(0)
            if not n:
                return ('SKIP', '没有非容器元素可作为 path')
            return {'path': n['path'], 'name': 'cli_probe_str', 'text': text}
        return build
    step('set_local_string:set', 'set_local_string', b_local_string(None, 'cli probe v1'))
    step('set_local_string:update', 'set_local_string', b_local_string(None, 'cli probe v2'))

    def b_set_items(c):
        """Prefer a NON_DATABASE ComboBox/RadioGroup.

        MEASURED (see the report's contract section): on a column-backed element (fieldType
        COLUMN_LIKE / TABLE_COLUMN) the `items` attribute `set_items` writes into the .4fd is
        REPOPULATED from the column's data-dictionary item list when the package is reloaded --
        RoundTrip.exe reports `stale` on exactly that attribute, while the in-session `verify`
        reports 0 because nothing has been reloaded. That is a fact about the write, not about this
        driver, and it is reported as such; driving the write through a column-backed element in
        the main sweep would only make the fixed-point verdict unreadable for everything else.
        """
        DB = ('COLUMN_LIKE', 'TABLE_COLUMN')

        def pick():
            for n in c.of_tag('ComboBox', 'RadioGroup'):
                _ok, r, _e = c.get(n['path'])
                if _ok and (r.get('layout') or {}).get('fieldType') not in DB:
                    return n
            return None

        c.ensure()
        n = pick()
        if n is None:
            # Either the file ships none, or every one it ships is column-backed. Adding a plain
            # one covers both, and the add is itself a real use of add_widget.
            host = b_in_container(c, ('Grid', 'Group', 'VBox', 'HBox', 'Table'))
            if not host:
                return ('SKIP', '文件里没有 ComboBox/RadioGroup 元素，也没有可用作父容器的元素')
            _i, ok, res, err = c.srv.call('add_widget', {'handle': c.h1, 'path': host['path'],
                                                         'type': 'ComboBox', 'name': 'cli_combo'})
            if not ok:
                return ('SKIP', '文件里没有可用的（非 column-backed）ComboBox/RadioGroup，'
                                '就地新建也失败: ' + short(err, 80))
            c.dirty = True
            c.ensure()
            n = pick()
            if n is None:
                return ('SKIP', '新建 ComboBox 后仍然找不到可写 items 的元素')
        return {'path': n['path'],
                'items': ['CLI_A', 'CLI_B|bee', 'CLI_C|cee|cli probe description']}
    step('set_items', 'set_items', b_set_items)

    def b_set_excluded(c):
        c.ensure()
        n = leaf_with_spec()
        if not n:
            return ('SKIP', '没有带规格节点的元素')
        return {'path': n['path'], 'excluded': True}
    step('set_excluded', 'set_excluded', b_set_excluded)

    def b_progrel(c):
        c.ensure()
        el = c.element_for_spec('pfield')
        if not el:
            return ('SKIP', '这张表单没有 pfield 规格节点（或没有任何元素的 FormSpeDictionary 条目持有它）')
        ok, res, _e =c.get(el['path'], kind='pfield')
        if not ok:
            return ('SKIP', 'get_component(kind=pfield) 失败: ' + short(res))
        attrs = ((res.get('spec') or {}).get('pfield') or {}).get('attrs') or {}
        if attrs.get('cite_std') != 'N':
            return ('SKIP', 'pfield 的 cite_std=%r（设计器的串查面板在 CiteStd 非 N 时整体禁用）'
                    % attrs.get('cite_std'))
        return {'path': el['path'], 'program': 'cli_probe_prog'}
    step('set_progrel_programs', 'set_progrel_programs', b_progrel)

    def b_table_assoc(c):
        c.ensure()
        tbls, srs = tsd_table_info(c.target)
        el = None
        for nm in srs:
            n = c.named(nm)
            if n and n['tag'] != 'Form':
                el = n
                break
        if el is None:
            for n in c.N:
                if n['name'] and n['tag'] != 'Form':
                    el = n
                    break
        if el is None:
            return ('SKIP', '没有带 name 的元素，<sr name=…> 无从对应')
        return {'path': el['path'], 'table': tbls[0] if tbls else 'cli_probe_no_such_table'}
    step('set_table_association:table', 'set_table_association', b_table_assoc)

    def b_table_flag(c):
        # SRAttributesCB_Click's branch: the checkbox edits an <sr> that already exists and never
        # creates one, so this only has anything to do when the move above (or the file) left one.
        c.ensure()
        _t, srs = tsd_table_info(c.target)
        for nm in srs:
            n = c.named(nm)
            if n and n['tag'] != 'Form':
                return {'path': n['path'], 'table': 'unused', 'column': 'insert'}
        return ('SKIP', '这张 .tsd 里没有 <sr name=…> 行（SRAttributesCB_Click 只改已存在的行）')
    step('set_table_association:flag', 'set_table_association', b_table_flag)

    def b_cited(c):
        c.ensure()
        n = leaf_with_spec()
        if not n:
            return ('SKIP', '没有带规格节点的元素')
        return {'path': n['path'], 'cited': True}
    step('set_cited', 'set_cited', b_cited)

    # ---- 属性: set_tree_source / rename_component ----------------------------------------------

    def b_tree_source(c):
        c.ensure()
        # The element this needs is the one whose FormSpecModel holds a <tree> NODE -- not the one
        # whose tag is Tree. In aist310_wf the <tree> spec node belongs to the Table `s_browse`,
        # and neither Tree-tagged element has one (Attr.Fsm on them returns null, which is exactly
        # why the naive choice answers E_DESIGNER instead of the not_found the message promises).
        el = c.element_for_spec('tree')
        if el is None:
            return ('SKIP', '没有任何元素的 FormSpeDictionary 条目持有 tree 规格节点'
                            '（数据来源只存在于带 <tree> 规格的组件上）')
        ok, res, _e = c.get(el['path'], kind='tree')
        if not ok:
            return ('SKIP', 'get_component(kind=tree) 失败: ' + short(res))
        kids = ((res.get('spec') or {}).get('tree') or {}).get('children') or {}
        if not kids:
            return ('SKIP', '%s 的 tree 规格节点没有子元素（无数据来源格子）' % el['name'])
        for elem, at in kids.items():
            for a, v in (at or {}).items():
                if a in ('table', 'col', 'src') and v:
                    return {'path': el['path'], 'element': elem, 'property': a, 'value': v + '_cli'}
        return ('SKIP', '%s 的 tree 规格节点里 table/col/src 三个属性都是空的，没有可改的值' % el['name'])

    step('set_tree_source', 'set_tree_source', b_tree_source)

    # ---- 结构: 新建 / 插入 / 删除 / 改名 --------------------------------------------------------

    def b_in_container(c, tag_pref=('Grid',)):
        c.ensure()
        for tg in tag_pref:
            for n in c.of_tag(tg):
                return n
        for n in c.N:
            if n['tag'] not in ('Form', 'Page'):
                return n
        return None

    def b_add_widget(c, typ, name):
        def build(cc):
            cc.ensure()
            host = b_in_container(cc, ('Grid', 'Group', 'VBox', 'HBox', 'Table'))
            if not host:
                return ('SKIP', '没有可用作父容器的元素')
            return {'path': host['path'], 'type': typ, 'name': name}
        return build
    step('add_widget:w1', 'add_widget', b_add_widget(None, 'TimeEdit', 'cli_w1'))
    step('add_widget:w2', 'add_widget', b_add_widget(None, 'Slider', 'cli_w2'))

    def b_insert_at(c):
        c.ensure()
        host = b_in_container(c, ('Grid', 'Group', 'VBox', 'HBox', 'Table'))
        if not host:
            return ('SKIP', '没有可用作父容器的元素')
        return {'path': host['path'], 'type': 'SpinEdit', 'index': 0}
    step('insert_at', 'insert_at', b_insert_at)

    def b_add_field(container=None):
        """`add_field` needs (table, column) whose col_attr carries a widget.

        The declared way to find one is list_columns -- which NREs today (see the contract probe),
        so the same fact is read from the workspace's own files instead: mta/tables.xml gives the
        table and its module, <ws>/<module>/tbl/<table>.tbl gives the col_attr widget mappings.
        That is the same data TableColumnHelper caches, read one level lower.
        """
        def build(cc):
            cc.ensure()
            host = None
            for n in cc.of_tag('Table'):
                host = n
                break
            if not host:
                host = b_in_container(cc, ('Grid', 'Group', 'VBox'))
            if not host:
                return ('SKIP', '没有可用作父容器的元素')
            pairs = widget_columns(cc)
            if not pairs:
                return ('SKIP', '工作区数据字典里找不到「列元数据带 col_attr@widget」的表/列'
                                '（list_tables + list_columns 都没有给出可用的列）')
            present = set(n['name'] for n in cc.N)
            out = []
            for t, col in pairs:
                if col in present or ('lbl_' + col) in present:
                    continue
                args = {'path': host['path'], 'table': t, 'column': col}
                if container:
                    args['container'] = container
                out.append(args)
                if len(out) >= 3:
                    break
            if not out:
                return ('SKIP', '工作区里所有带 widget 映射的列都已经在表单上')
            return out
        return build
    step('add_field', 'add_field', b_add_field(), candidates=True)
    # Two containers, so `wrap` has something to wrap whose PARENT is legal for break_layout.
    # No corpus file has a container that is itself a child of a non-Form/HBox/VBox parent AND
    # holds >= 2 container children, so the structure the wrap -> break_layout pair needs is
    # built here instead of hoped for.
    step('add_field:grp1', 'add_field', b_add_field('Group'), candidates=True)
    step('add_field:grp2', 'add_field', b_add_field('Group'), candidates=True)

    def b_delete(c):
        c.ensure()
        w = c.named('cli_w1')
        if not w or w['tag'] in CONTAINERS:
            return ('SKIP', 'add_widget 没有留下 cli_w1（上一步失败）')
        # `path` is required by the manifest even on the batch form (Struct.PathsArg).
        return {'path': w['path'], 'paths': [w['path']]}
    step('delete', 'delete', b_delete)

    def b_rename(c):
        c.ensure()
        w = c.named('cli_w2')
        if not w:
            return ('SKIP', 'add_widget 没有留下 cli_w2（上一步失败）')
        return {'path': w['path'], 'name': 'cli_w2_rn'}
    step('rename_component', 'rename_component', b_rename, dirty=True)

    # ---- 结构: 转换 / Z 序 / 对齐 / 尺寸 --------------------------------------------------------

    def b_convert_widget(c):
        c.ensure()
        targets = ['Edit', 'ButtonEdit', 'CheckBox', 'ComboBox', 'TextEdit']
        skip_ct = {'PROGREL', 'NONE', 'REFERENCE', 'MULTILANG'}
        for n in c.leaves():
            if n['tag'] not in ('ButtonEdit', 'CheckBox', 'ComboBox', 'Edit', 'DateEdit',
                                'TextEdit', 'ProgressBar', 'Slider', 'SpinEdit', 'TimeEdit'):
                continue
            ok, res, _e =c.get(n['path'])
            if not ok:
                continue
            st = res.get('specNodeType')
            if st in skip_ct or st is None:
                continue
            mine = res.get('tag')
            tgt = next((t for t in targets if t != mine), None)
            if tgt:
                return {'path': n['path'], 'type': tgt}
        return ('SKIP', '没有可转换的控件元素（需要非容器、有规格节点、且 SpecNodeType 不在 '
                        'PROGREL/NONE/REFERENCE/MULTILANG）')
    step('convert_widget', 'convert_widget', b_convert_widget, dirty=True)

    def b_convert_container(c):
        c.ensure()
        for tg, other in (('Grid', 'Group'), ('Group', 'Grid')):
            for n in c.of_tag(tg):
                return {'path': n['path'], 'type': other}
        return ('SKIP', '文件里没有 Grid/Group 容器（CanExecuteConvertToContainer 要求源元素本身是容器）')
    step('convert_container', 'convert_container', b_convert_container, dirty=True)

    def b_move(c):
        c.ensure()
        for box in c.of_tag('Folder', 'HBox', 'VBox', 'Table', 'Tree'):
            kids = [n for n in c.N if n['parent'] == box['path']]
            if len(kids) >= 2:
                return {'paths': [kids[1]['path']], 'to': 'first'}
        return ('SKIP', '没有 Folder/HBox/VBox/Table/Tree 父容器带 >= 2 个直接子节点（设计器的 canMoveInBox 门禁）')
    step('move', 'move', b_move, dirty=True)

    def b_align(c):
        c.ensure()
        by_parent = {}
        for n in c.leaves():
            by_parent.setdefault(n['parent'], []).append(n)
        for _p, kids in by_parent.items():
            if len(kids) >= 2:
                return {'paths': [kids[0]['path'], kids[1]['path']], 'option': 'left'}
        return ('SKIP', '找不到同一父节点下的 2 个非容器元素（设计器 CanExecuteAlignWidgets 要求多于 1 个）')
    step('align', 'align', b_align)

    def b_fit(c):
        c.ensure()
        n = pick_leaf(0)
        if not n:
            return ('SKIP', '没有非容器元素')
        return {'paths': [n['path']]}
    step('fit_size', 'fit_size', b_fit)

    def b_nudge(c):
        c.ensure()
        n = pick_leaf(0)
        if not n:
            return ('SKIP', '没有非容器元素')
        return {'paths': [n['path']], 'direction': 'right', 'offset': 1}
    step('nudge', 'nudge', b_nudge)

    # ---- 页签 ----------------------------------------------------------------------------------

    def b_add_page(c):
        c.ensure()
        folds = c.of_tag('Folder')
        if not folds:
            return ('SKIP', '文件里没有 Folder 元素（Page 是 Folder 的专属子节点）')
        return {'path': folds[0]['path'], 'name': 'cli_probe_page'}
    step('add_page', 'add_page', b_add_page, dirty=True)

    def b_delete_page(c):
        c.ensure()
        pg = c.named('cli_probe_page')
        if not pg:
            return ('SKIP', 'add_page 没有留下 cli_probe_page（上一步失败）')
        return {'path': pg['path']}
    step('delete_page', 'delete_page', b_delete_page, dirty=True)

    # ---- 语义 / Action -------------------------------------------------------------------------

    def b_ins_sem(c, use):
        def build(cc):
            cc.ensure()
            if use == 'multilang':
                host = b_in_container(cc, ('Table', 'Grid', 'Group', 'VBox', 'HBox'))
            else:
                host = None
                for tg in ('Table', 'Tree'):
                    for n in cc.of_tag(tg):
                        host = n
                        break
                    if host:
                        break
                if not host:
                    return ('SKIP', '%s 栏位只能加在 Table/Tree 里，文件里没有' % use)
            if not host:
                return ('SKIP', '没有可用作父容器的元素')
            return {'path': host['path'], 'use': use}
        return build
    step('insert_semantic:multilang', 'insert_semantic', b_ins_sem(None, 'multilang'))
    step('insert_semantic:reference', 'insert_semantic', b_ins_sem(None, 'reference'))
    step('insert_semantic:progrel', 'insert_semantic', b_ins_sem(None, 'progrel'))

    def b_add_action(c):
        c.ensure()
        # standalone: path="" is the only value a required `path` admits that can never be a
        # name-path, and it is exactly ActionDefaults.ExecutedAddAction. Works on every file.
        return {'path': '', 'type': 'all', 'name': 'cli_probe_act'}
    step('add_action:standalone', 'add_action', b_add_action)

    def b_add_action_bound(c):
        c.ensure()
        for n in c.of_tag('Button'):
            ok, res, _e =c.get(n['path'])
            if ok and res.get('specNodeType') == 'ACTION':
                return {'path': n['path'], 'type': 'all'}
        return ('SKIP', '文件里没有 SpecNodeType==ACTION 的 Button（只有 Button 且 style 非 button_qrystr 才是）')
    step('add_action:bound', 'add_action', b_add_action_bound)

    step('set_action_types:mi', 'set_action_types',
         lambda c: {'id': 'cli_probe_act', 'types': 'mi'})
    step('set_action_types:none', 'set_action_types',
         lambda c: {'id': 'cli_probe_act', 'types': 'none'})
    step('delete_action', 'delete_action', lambda c: {'id': 'cli_probe_act'})

    # ---- Tab 顺序 ------------------------------------------------------------------------------

    def b_tab_action(c):
        c.ensure()
        for n in c.leaves():
            ok, res, _e =c.get(n['path'])
            if ok and 'tabIndex' in (res.get('layout') or {}):
                return {'paths': [n['path']], 'action': 'first'}
        return ('SKIP', '文件里没有任何元素带 tabIndex（只有 mod-fd.spec 里列了 tabIndex 的十几种才有）')
    step('tab_action:first', 'tab_action', b_tab_action)

    def b_tab_action_auto(c):
        c.ensure()
        for n in c.leaves():
            ok, res, _e =c.get(n['path'])
            if ok and 'tabIndex' in (res.get('layout') or {}):
                return {'paths': [n['path']], 'action': 'auto'}
        return ('SKIP', '文件里没有任何元素带 tabIndex')
    step('tab_action:auto', 'tab_action', b_tab_action_auto)

    step('set_tab_order:document', 'set_tab_order', lambda c: {'paths': []})

    def b_set_tab_order_first(c):
        c.ensure()
        picks = []
        for n in c.leaves():
            ok, res, _e =c.get(n['path'])
            if ok and 'tabIndex' in (res.get('layout') or {}):
                picks.append(n['path'])
            if len(picks) == 2:
                return {'paths': picks}
        return ('SKIP', '不足 2 个带 tabIndex 的元素')
    step('set_tab_order:first', 'set_tab_order', b_set_tab_order_first)

    # ---- 校验 / 工具 ---------------------------------------------------------------------------

    def b_code_template(c):
        c.ensure()
        # The workspace's own code_template.xml is the authority (SetCodeTemplate reads it).
        ws = c.ws
        f = os.path.join(ws, 'mta', 'code_template.xml')
        ids = []
        if os.path.exists(f):
            try:
                txt = open(f, 'r', encoding='utf-8', errors='replace').read()
                import re
                ids = re.findall(r'<kind\s+id="([^"]+)"', txt)
            except Exception:
                ids = []
        if len(ids) < 2:
            return ('SKIP', 'workspace 的 mta/code_template.xml 里 <kind id=…> 不足 2 个，没有可切换的值')
        n = pick_leaf(0)
        if not n:
            return ('SKIP', '没有非容器元素可作为 path')
        cur = c.tree.get('codeTemplate')
        tgt = next((i for i in ids if i != cur), ids[1])
        return {'path': n['path'], 'template': tgt}
    step('set_code_template', 'set_code_template', b_code_template)

    # ---- wrap / break_layout: LAST, because they move other elements' paths --------------------

    def b_wrap(c):
        c.ensure()
        # The designer's own mime table (modFD/HBox) accepts Folder/Grid/Group/HBox/ScrollGrid/
        # Table/Tree/VBox and NOTHING else -- a Page or a plain widget is refused by it, and that
        # refusal is the module's negative control, not the positive path.
        ok_children = {'Folder', 'Grid', 'Group', 'HBox', 'ScrollGrid', 'Table', 'Tree', 'VBox'}
        out = []
        # (1) the two containers add_field built: their parent is a Grid, so the box wrap makes
        #     lands in a legal parent and break_layout can undo it -- the dependency the task asks
        #     for, which no corpus file's own structure can express (see the add_field:grp step).
        made = []
        for sid in ('add_field:grp1', 'add_field:grp2'):
            r = c.res.get(sid)
            if not r:
                continue
            for a in (r.get('added') or []):
                if a.get('tag') in CONTAINERS and a.get('path') and c.node(a['path']):
                    made.append(a['path'])
                    break
        if len(made) >= 2:
            out.append({'paths': made[:2], 'type': 'hbox'})
        # (2) a container the file already has whose own parent is legal for break_layout.
        for box in c.N:
            if box['tag'] not in CONTAINERS or box['tag'] == 'Form':
                continue
            par = c.parent(box)
            if not par or par['tag'] in ('Form', 'HBox', 'VBox'):
                continue
            kids = [n for n in c.N if n['parent'] == box['path']]
            cont = [k for k in kids if k['tag'] in ok_children]
            if len(cont) >= 2:
                out.append({'paths': [cont[0]['path'], cont[1]['path']], 'type': 'hbox'})
                break
        # (3) anything wrappable, even if break_layout will then refuse the result.
        for box in c.N:
            if box['tag'] not in CONTAINERS or box['tag'] == 'Form':
                continue
            kids = [n for n in c.N if n['parent'] == box['path']]
            cont = [k for k in kids if k['tag'] in ok_children]
            if len(cont) >= 2:
                out.append({'paths': [cont[0]['path'], cont[1]['path']], 'type': 'hbox'})
                break
        if not out:
            return ('SKIP', '找不到「父容器下 >= 2 个容器子节点」的结构'
                            '（wrap 的 mime 表只收容器，不收控件/Page）')
        return out
    step('wrap', 'wrap', b_wrap, dirty=True, candidates=True)

    def b_break_wrapped(c):
        # The dependency the task asks for: break_layout is only meaningful on a box wrap made.
        c.ensure()
        r = c.res.get('wrap')
        if not r:
            return ('SKIP', 'wrap 没有成功，没有可拆的盒子')
        p = r.get('path')
        if not p:
            return ('SKIP', 'wrap 的返回体里没有 path')
        return {'path': p}
    step('break_layout:wrapped', 'break_layout', b_break_wrapped, dirty=True)

    def b_break_natural(c):
        c.ensure()
        for n in c.of_tag('HBox', 'VBox'):
            par = c.parent(n)
            if par and par['tag'] not in ('Form', 'HBox', 'VBox'):
                return {'path': n['path']}
        return ('SKIP', '没有父容器不是 Form/HBox/VBox 的 HBox/VBox（CanBreakLayout 的门禁）')
    step('break_layout:natural', 'break_layout', b_break_natural, dirty=True)

    # `copy_component` used to be the last step here. It was withdrawn from the product: it never
    # copied (same-form it is a cut+paste that lands the elements back in the SAME container, i.e.
    # a reorder that `move {to:"last"}` expresses better; cross-form `TargetContainer` falls back
    # to the target's <Form> and `CanExecutePaste` refuses it), and it emptied the source form on
    # the way. It is E_UNKNOWN_METHOD now, so driving it would only manufacture an E_* that is not
    # a write verdict. Removed from WRITE_FNS and from the negative controls with it.

    return P


# ================================================================================= negative controls
#
# ONE PER WRITE GROUP (the task's rule), plus extras where they are cheap. Each must come back as
# ok:false with a declared code, and must leave the package byte-identical.

def negatives(ctx):
    ctx.ensure()
    N = []

    def add(group, fn, args):
        N.append((group, fn, args))

    # 属性 -- a layout attribute the element does not have (the whitelist-by-presence rule)
    def leaf():
        ctx.ensure()
        ls = ctx.leaves()
        return ls[0]['path'] if ls else None

    def spec_el(kind='field'):
        ctx.ensure()
        e = ctx.element_for_spec(kind)
        return e['path'] if e else None

    lf = leaf()
    if lf:
        add('属性', 'set_layout_attr', {'path': lf, 'attr': 'zzNotARealAttribute', 'value': '1'})
    sp = spec_el()
    if sp:
        add('属性', 'set_spec_attr', {'path': sp, 'kind': 'field',
                                      'attr': 'zzNotARealAttr', 'value': '1'})
    if lf:
        add('属性', 'rename_component', {'path': lf, 'name': 'bad name!'})
    ts = ctx.element_for_spec('tree')
    if ts:
        add('属性', 'set_tree_source', {'path': ts['path'], 'element': 'zz_no_such_cell',
                                        'property': 'table', 'value': 'x'})

    # 结构
    if lf:
        add('结构', 'nudge', {'paths': [lf], 'direction': 'sideways'})
        add('结构', 'align', {'paths': [lf], 'option': 'left'})
        add('结构', 'move', {'paths': [lf], 'to': 'sideways'})
        add('结构', 'wrap', {'paths': [lf], 'type': 'hbox'})       # widget into HBox -> mime gate
        add('结构', 'break_layout', {'path': lf})                  # not an HBox/VBox
        add('结构', 'convert_widget', {'path': lf, 'type': 'Button'})   # not in the 14 targets
        add('结构', 'convert_container', {'path': lf, 'type': 'Group'})  # not a Grid/Group
        add('结构', 'add_field', {'path': lf, 'table': 'zzz_t', 'column': 'zzz001',
                                  'container': 'zzz'})             # enum, manifest-level
        add('结构', 'add_widget', {'path': lf, 'type': 'NotAWidget'})    # enum, manifest-level
        add('结构', 'insert_at', {'path': lf, 'type': 'Edit', 'index': -1})  # index out of range

    # 复制 -- there is no longer such a group: copy_component was withdrawn from the product
    # (see the plan()). The 8 remaining groups each still carry at least one negative.

    # 页签
    if lf:
        add('页签', 'add_page', {'path': lf})          # not a Folder/Page
        add('页签', 'delete_page', {'path': lf})       # not a Page

    # 语义 / Action
    add('语义/Action', 'delete_action', {'id': 'zz_no_such_action'})
    add('语义/Action', 'set_action_types', {'id': 'zz_no_such_action', 'types': 'all'})
    if lf:
        add('语义/Action', 'add_action', {'path': lf, 'type': 'all'})   # SpecNodeType != ACTION
        add('语义/Action', 'insert_semantic', {'path': lf, 'use': 'bogus'})  # enum

    # 多语言 / 选项 / 串查
    if lf:
        add('多语言/选项/串查', 'set_items', {'path': lf, 'items': ['X']})  # not a ComboBox
        add('多语言/选项/串查', 'set_progrel_programs', {'path': lf, 'program': 'X'})  # no pfield
        add('多语言/选项/串查', 'set_table_association', {'path': lf, 'table': 'zz_no_such_table'})
        add('多语言/选项/串查', 'set_spec_description', {'path': lf, 'kind': 'zzz', 'content': 'x'})
        add('多语言/选项/串查', 'set_local_string', {'path': 'zz/no/such/path', 'name': 'x',
                                                     'text': 'y'})

    # Tab 顺序
    if lf:
        add('Tab 顺序', 'tab_action', {'paths': [lf], 'action': 'bogus'})
        add('Tab 顺序', 'set_tab_order', {'paths': [lf]})    # element without tabIndex -> designer

    # 校验 / 工具
    add('校验/工具', 'set_excluded', {'path': 'zz/no/such/path', 'excluded': True})
    if lf:
        add('校验/工具', 'set_code_template', {'path': lf, 'template': 'Z'})

    # 会话
    add('会话', 'close', {'handle': 12345})       # handle must be a string

    return N


# ================================================================================= the per-file run

def one_file(target, idx, buddy, timeout, only_fns):
    ws = g3.ws_of(target)
    tag = os.path.basename(target)
    d = os.path.dirname(target)
    outs = dict((k, os.path.join(d, '_ai_w3fns_%d_%s.tzs' % (idx, k)))
                for k in ('pre', 'mid', 'post'))
    for p in outs.values():
        if os.path.exists(p):
            os.remove(p)

    say('')
    say('=======================================================================')
    say('%s   ws=%s' % (tag, ws))

    pf, pline = g3.roundtrip(target, ws)
    base = g3.rt_parse(pf)
    if not base:
        say('  !! pristine RoundTrip did not answer: %s' % (pline or 'no SUMMARY')[:200])
        return None
    say('  pristine RoundTrip: %s' % pline[:170])

    srv = Server(ws, str(idx))
    ctx = Ctx(srv, target, ws, idx)
    result = {'tag': tag, 'ws': ws, 'base': base, 'verdicts': {}, 'notes': []}

    try:
        i, ok, res, err = srv.call('open', {'path': target})
        if not ok:
            say('  !! open failed: ' + short(err))
            result['notes'].append('open 失败')
            return result
        ctx.h1 = res.get('handle')
        say('  open -> %s  (%s)' % (ctx.h1, res.get('key')))

        # A SECOND handle on a different package, in the same process.
        #
        # copy_component used to be its only consumer and is gone from the product, so this had to
        # be re-justified or deleted. It is kept, because it is the only place in this repo that
        # exercises SPEC §11.24 (g) end to end while real work is happening:
        #
        #   * the registry must hold two live handles with two DIFFERENT ProgramKeys at once --
        #     §11.24 (g) exists because four corpus files share one key, and the address is the
        #     handle, never the key;
        #   * the sweep on h1 must not flip h2's `mutated`. That flag is set by RegisterUndoRedo,
        #     i.e. precisely the per-key state §11.24 (j) warns can bleed between two handles open
        #     at the same time ("_specDic/_infoCount 模块静态：两个 handle 同时开会串扰");
        #   * every file-level verdict below (validate, RoundTrip, the byte comparisons) is then
        #     produced with a FOREIGN package resident, which is strictly stronger evidence than
        #     producing them alone.
        #
        # Both points are asserted (F9/F10). If that assertion is not wanted, delete this whole
        # block and `buddy_for` -- nothing else depends on it.
        if buddy:
            j, ok2, res2, err2 = srv.call('open', {'path': buddy})
            if ok2:
                ctx.h2 = res2.get('handle')
                say('  open(buddy) -> %s  (%s)' % (ctx.h2, os.path.basename(buddy)))
                _i, lok, lres, lerr = srv.call('list_open', {})
                rows = lres if isinstance(lres, list) else []
                keys = sorted(set(r.get('key') for r in rows if r.get('handle')))
                result['twoHandles'] = len(rows) >= 2 and len(keys) >= 2
                say('    list_open -> %d handles, keys=%s' % (len(rows), keys))
            else:
                result['twoHandles'] = False
                result['notes'].append('buddy open 失败: ' + short(err2, 90))
                say('  buddy open failed: ' + short(err2, 120))

        # ---- reads -------------------------------------------------------------------------
        say('  -- reads')
        for k in g3.KINDS:
            _i, k_ok, k_res, k_err = srv.call('describe_kind', {'handle': ctx.h1, 'kind': k})
            n = len(k_res.get(k) or []) if k_ok and isinstance(k_res.get(k), list) else None
            ctx.kinds[k] = n
            result.setdefault('kinds', {})[k] = n
        _i, ok, res, err = srv.call('list_spec_nodes', {'handle': ctx.h1})
        ctx.spec_nodes = res.get('nodes') or []
        result['specNodes'] = len(ctx.spec_nodes)
        _i, ok, res, err = srv.call('list_local_strings', {'handle': ctx.h1})
        ctx.strings = res.get('count')
        ctx.ensure()
        result['elements'] = ctx.tree.get('elementCount')
        result['codeTemplate'] = ctx.tree.get('codeTemplate')
        say('    elements=%s codeTemplate=%s specNodes=%s strings=%s'
            % (result['elements'], result['codeTemplate'], result['specNodes'], ctx.strings))
        say('    describe_kind: ' + json.dumps(ctx.kinds, ensure_ascii=False))

        # A handle-less read the manifest declares: both NRE today. Recorded here because the
        # verdict is about the contract, not about the write surface.
        _i, ok_t, res_t, err_t = srv.call('list_tables', {})
        result['listTables'] = (ok_t, (err_t or {}).get('code'))
        _i, ok_c, res_c, err_c = srv.call('list_columns', {'table': 'pmdl_t'})
        result['listColumns'] = (ok_c, (err_c or {}).get('code'))

        # ---- validate(baseline) ------------------------------------------------------------
        _i, ok, res, err = srv.call('validate', {'handle': ctx.h1}, timeout=max(timeout, 300))
        if not ok:
            say('  !! validate(baseline) failed: ' + short(err))
            result['notes'].append('validate 基线失败')
            return result
        result['baseErr'] = len([e for e in (res.get('after') or []) if e.get('type') == 'ERROR'])
        result['baseWarn'] = len([e for e in (res.get('after') or []) if e.get('type') == 'WARNING'])
        say('  validate(baseline): after=%d ERROR / %d WARNING, %sms'
            % (result['baseErr'], result['baseWarn'], res.get('elapsedMs')))

        # ---- save S_pre --------------------------------------------------------------------
        _i, ok, res, err = srv.call('save', {'handle': ctx.h1, 'out': outs['pre']})
        if not ok:
            say('  !! save(pre) failed: ' + short(err))
            return result
        say('  save(pre)  -> %d bytes' % os.path.getsize(outs['pre']))

        # ---- negative probes ---------------------------------------------------------------
        say('  -- negative probes (must be refused, must change zero bytes)')
        for group, fn, args in negatives(ctx):
            if args is None:
                continue
            args = dict(args)
            args.setdefault('handle', ctx.h1)
            _i, nok, nres, nerr = srv.call(fn, args)
            code = (nerr or {}).get('code')
            good = (not nok) and code in REFUSAL_CODES
            neg_results.append((tag, group, fn, code, good))
            say('    %-6s %-24s %-34s -> %s' % ('ok' if good else '!!', fn, group,
                                               code if not nok else ('OK:true ' + short(nres, 70))))

        # ---- save S_mid, byte-compare ------------------------------------------------------
        _i, ok, res, err = srv.call('save', {'handle': ctx.h1, 'out': outs['mid']})
        if not ok:
            say('  !! save(mid) failed: ' + short(err))
            return result
        same = g3.sha(outs['pre']) == g3.sha(outs['mid'])
        result['negBytesSame'] = same
        say('  save(mid)  -> %s  (bytes identical to pre: %s)'
            % (os.path.getsize(outs['mid']), same))

        # ---- the write sweep ---------------------------------------------------------------
        say('  -- write sweep')
        P = plan(ctx)
        for st in P:
            fn = st['fn']
            if only_fns and fn not in only_fns:
                continue
            try:
                b = st['build'](ctx)
            except Exception as e:
                result['verdicts'].setdefault(fn, []).append('%s: FAIL(harness build: %s)'
                                                             % (st['id'], e))
                say('    !! %-28s harness build error: %s' % (st['id'], e))
                continue
            if isinstance(b, tuple) and b and b[0] == 'SKIP':
                skips.setdefault((tag, fn), b[1])
                result['verdicts'].setdefault(fn, []).append('%s: SKIP' % st['id'])
                say('    -- %-28s SKIP: %s' % (st['id'], b[1]))
                continue
            calls = b if isinstance(b, list) else [b]
            for c in calls:
                c = dict(c)
                c.setdefault('handle', ctx.h1)
                _i, wok, wres, werr = srv.call(fn, c, timeout=max(timeout, 300))
                v, why = verdict(wok, wres, werr)
                line = '%s: %s(%s)' % (st['id'], v, why)
                result['verdicts'].setdefault(fn, []).append(line)
                if wok:
                    ctx.res[st['id'].split(':')[0]] = wres
                say('    %-5s %-28s %-52s %s' % (v, st['id'], short(c, 52),
                                                 why + (' :: ' + ((werr or {}).get('message') or '')[:90]
                                                        if v in ('REFUSED',) else '')))
                if v == 'FAIL':
                    say('          detail: ' + note_of(wok, wres, werr))
                if wok:
                    ctx.res[st['id']] = wres
                    if st.get('candidates'):
                        break
            if st.get('dirty') or fn in DIRTY_FNS:
                ctx.dirty = True
            ctx.ensure()

        # ---- validate(after) ---------------------------------------------------------------
        _i, ok, res, err = srv.call('validate', {'handle': ctx.h1}, timeout=max(timeout, 300))
        result['newErrors'] = None
        if not ok:
            say('  !! validate(after) failed: ' + short(err))
        else:
            ne = res.get('newErrors') or []
            nw = res.get('newWarnings') or []
            result['newErrors'] = len(ne)
            result['newWarnings'] = len(nw)
            say('  validate(after): newErrors=%d newWarnings=%d' % (len(ne), len(nw)))
            for e in ne[:6]:
                say('      NEW ERR: ' + short(e, 160))

        # ---- save S_post + RoundTrip -------------------------------------------------------
        _i, ok, res, err = srv.call('save', {'handle': ctx.h1, 'out': outs['post']})
        if not ok:
            say('  !! save(post) failed: ' + short(err))
            return result
        result['postBytes'] = os.path.getsize(outs['post'])
        result['wroteSomething'] = (g3.sha(outs['pre']) != g3.sha(outs['post']))
        say('  save(post) -> %d bytes (differs from pre: %s)'
            % (result['postBytes'], result['wroteSomething']))

        f, ln = g3.roundtrip(outs['post'], ws)
        rt = g3.rt_parse(f)
        result['rt'] = rt
        result['rtLine'] = ln
        result['rtClean'] = g3.rt_ok(rt, base)
        say('  RoundTrip(post): %s' % ln[:180])
        say('  fixed point (baseline-relative): %s' % result['rtClean'])

        # ---- post-verdict probe: set_items on a COLUMN-BACKED element ------------------------
        # Runs AFTER S_post on purpose: it mutates the session model, and the verdict above must
        # describe what the sweep produced, not this one. Its own save is thrown away.
        result['dbItems'] = db_items_probe(ctx, outs['post'].replace('.tzs', '_dbprobe.tzs'))
        if result['dbItems']:
            nm, pok, rtstale, rsample = result['dbItems']
            say('  [probe] set_items on column-backed "%s": ok=%s  RoundTrip(reload) stale=%s'
                % (nm, pok, rtstale))
            if rsample:
                say('          %s' % rsample[:150])

        # The foreign handle must have been left alone by everything above (see the buddy comment).
        if ctx.h2:
            _i, lok, lres, lerr = srv.call('list_open', {})
            rows = lres if isinstance(lres, list) else []
            row = next((r for r in rows if r.get('handle') == ctx.h2), None)
            result['h2Clean'] = bool(row) and row.get('mutated') is False
            say('  list_open -> h2 alive=%s mutated=%s'
                % (bool(row), row.get('mutated') if row else None))

        # ---- close -------------------------------------------------------------------------
        for h in (ctx.h2, ctx.h1):
            if not h:
                continue
            _i, cok, cres, cerr = srv.call('close', {'handle': h})
            v, why = verdict(cok, cres, cerr)
            if h == ctx.h1 and not only_fns:
                result['verdicts'].setdefault('close', []).append('close: %s(%s)' % (v, why))
            say('  close(%s) -> %s (%s)' % (h, v, why))
        ctx.h1 = None
        ctx.h2 = None
    finally:
        srv.close()

    return result


def db_items_probe(ctx, tmp):
    """`set_items` on a column-backed ComboBox/RadioGroup, judged by a reload.

    MEASURED BEHAVIOUR: the `items` attribute the function splices into the .4fd is repopulated
    from the column's data-dictionary item list on reload, so RoundTrip.exe reports `stale` on it
    while the live session shows nothing wrong (Verify compares the model against the PRISTINE file
    on disk, so it cannot see this at all). Returns (name, ok, stale, sample) or None when the file
    ships no column-backed ComboBox/RadioGroup.
    """
    DB = ('COLUMN_LIKE', 'TABLE_COLUMN')
    el = None
    for n in ctx.of_tag('ComboBox', 'RadioGroup'):
        _ok, r, _e = ctx.get(n['path'])
        if _ok and (r.get('layout') or {}).get('fieldType') in DB:
            el = n
            break
    if el is None:
        return None
    if os.path.exists(tmp):
        os.remove(tmp)
    _i, ok, res, err = ctx.srv.call('set_items', {'handle': ctx.h1, 'path': el['path'],
                                                  'items': ['ZZ1|z1', 'ZZ2|z2']})
    if not ok:
        return (el['name'], False, None, None)
    _i, sok, sres, serr = ctx.srv.call('save', {'handle': ctx.h1, 'out': tmp})
    stale = None
    sample = ''
    if sok and os.path.exists(tmp):
        _f, ln = g3.roundtrip(tmp, ctx.ws)
        p = g3.rt_parse(_f)
        if p:
            stale = p['stale']
            sample = p.get('sample') or ''
        try:
            os.remove(tmp)
        except OSError:
            pass
    return (el['name'], True, stale, sample)


# ================================================================================= main

def main():
    argv = sys.argv[1:]
    limit = 0
    only = None
    root = ROOT
    keep = False
    timeout = 300
    only_fns = None
    i = 0
    while i < len(argv):
        a = argv[i]
        if a == '--limit':
            i += 1
            limit = int(argv[i])
        elif a == '--files':
            i += 1
            only = [x for x in argv[i].split(',') if x]
        elif a == '--root':
            i += 1
            root = argv[i]
        elif a == '--only-fn':
            i += 1
            only_fns = set(x for x in argv[i].split(',') if x)
        elif a == '--timeout':
            i += 1
            timeout = int(argv[i])
        elif a == '--keep':
            keep = True
        i += 1

    say('==== W3-FNS GATE ====  bin=%s' % BIN)
    say('corpus root: %s        %s' % (root, time.strftime('%Y-%m-%d %H:%M:%S')))

    files = g3.corpus(root)
    if only:
        files = [f for f in files if any(x in os.path.basename(f) for x in only)]
    if limit:
        files = files[:limit]
    say('corpus: %d files -> %s' % (len(files), ', '.join(os.path.basename(f) for f in files)))

    # The second handle (see the long comment in one_file): another corpus file in the SAME
    # workspace with a DIFFERENT program name -- a same-name file would be refused E_KEY_IN_USE by
    # design, which is itself the point of §11.24 (g).
    # (xiyuan/tst was cleared out of the workspace on 2026-09-19; its three names went with it, so
    # the second workspace is now xiyuan/prd. A missing buddy would have silently cost F9/F10 --
    # buddy_for returns None and the block is skipped -- which is why the names are listed.)
    buddies = {
        'hengshuo/prd': ['aapp320(c).tzs', 'cpmp530(c).tzs', 'aist310_wf(c).tzs'],
        'xiyuan/prd': ['abmm200_wf(c).tzs', 'ainq120_wf(c).tzs', 'asft310_wf(c).tzs',
                       'asft330(c).tzs'],
    }

    def buddy_for(target):
        b = os.path.basename(target)
        for key, names in buddies.items():
            if key.replace('/', os.sep) in target:
                for nm in names:
                    if nm != b:
                        p = os.path.join(os.path.dirname(target), nm)
                        if os.path.exists(p):
                            return p
        return None

    results = []
    t0 = time.time()
    for idx, f in enumerate(files):
        try:
            r = one_file(f, idx, buddy_for(f), timeout, only_fns)
        except subprocess.TimeoutExpired:
            say('  !! TIMEOUT')
            r = {'tag': os.path.basename(f), 'notes': ['TIMEOUT']}
        except Exception as e:
            say('  !! harness exception: %s: %s' % (type(e).__name__, e))
            r = {'tag': os.path.basename(f), 'notes': ['%s: %s' % (type(e).__name__, e)]}
        if r:
            results.append(r)
    say('')
    say('%d files in %.1f s' % (len(results), time.time() - t0))

    # ---- aggregate checks ------------------------------------------------------------------
    say('')
    say('--- aggregate')

    def agg(n, label, pred, extra=lambda r: ''):
        bad = [r for r in results if not pred(r)]
        g3.check(n, label, bool(results) and not bad,
                 (('%d bad: ' % len(bad)) + '; '.join('%s%s' % (r['tag'], extra(r))
                                                      for r in bad[:5])) if bad
                 else '%d files' % len(results))

    agg('F1', 'every file opened and was driven', lambda r: r.get('elements'))
    agg('F2', 'all 7 describe_kind answer with a non-empty vocabulary',
        lambda r: all((r.get('kinds') or {}).get(k) for k in g3.KINDS),
        lambda r: ' ' + json.dumps(r.get('kinds'), ensure_ascii=False)[:110])
    agg('F3', 'the refused writes changed ZERO bytes (sha256 pre == mid)',
        lambda r: r.get('negBytesSame') is True,
        lambda r: ' pre/mid differ')
    agg('F4', 'the write sweep changed something (sha256 pre != post)',
        lambda r: r.get('wroteSomething') is True,
        lambda r: ' pre==post: every write was a silent no-op')
    agg('F5', 'validate after the sweep reports zero NEW errors',
        lambda r: r.get('newErrors') == 0,
        lambda r: ' newErrors=%s newWarnings=%s' % (r.get('newErrors'), r.get('newWarnings')))
    agg('F6', 'the repacked file is a fixed point relative to its own pristine RoundTrip',
        lambda r: r.get('rtClean') is True,
        lambda r: ' ' + (r.get('rtLine') or '')[:140])
    agg('F7', 'every declared refusal carried a contract code',
        lambda r: all(g for (_t, _g, _f, _c, g) in neg_results if _t == r['tag']),
        lambda r: ' undeclared code: %s' % [(f, c) for (t, _g, f, c, g) in neg_results
                                            if t == r['tag'] and not g])

    def all_lines(r):
        out = []
        for _fn, lines in (r.get('verdicts') or {}).items():
            out.extend(lines)
        return out

    agg('F8', 'no write invocation produced an undeclared code or an exception',
        lambda r: not any(': FAIL(' in ln for ln in all_lines(r)),
        lambda r: ' ' + ' | '.join(ln for ln in all_lines(r) if ': FAIL(' in ln)[:220])
    agg('F9', 'two handles with two distinct ProgramKeys coexist (SPEC 11.24 (g))',
        lambda r: r.get('twoHandles') is True,
        lambda r: ' list_open did not show two distinct keys')
    agg('F10', 'the sweep on h1 left the other handle unmutable (no cross-package bleed)',
        lambda r: r.get('h2Clean') is True,
        lambda r: ' the buddy handle was mutated by work on h1')

    # ---- coverage --------------------------------------------------------------------------
    say('')
    say('--- coverage of the 33 mutating functions (CHANGED / NOOP / REFUSED / FAIL / SKIP)')
    hdr = ['function'.ljust(24)] + [r['tag'][:20].ljust(21) for r in results]
    say('    ' + ' '.join(hdr))
    covered = set()
    failed_fns = []
    for fn in WRITE_FNS:
        row = [fn.ljust(24)]
        any_fail = False
        for r in results:
            lines = (r.get('verdicts') or {}).get(fn, [])
            tag = r['tag']
            if not lines:
                row.append(('SKIP' if (tag, fn) in skips else '-').ljust(21))
                continue
            kinds = set()
            for ln in lines:
                v = ln.split(': ', 1)[1].split('(', 1)[0]
                kinds.add(v)
                if v != 'SKIP':
                    covered.add(fn)
                if v == 'FAIL':
                    any_fail = True
            row.append(('/'.join(sorted(kinds))).ljust(21))
        if any_fail:
            failed_fns.append(fn)
        say('    ' + ' '.join(row))

    say('')
    say('covered (>=1 non-SKIP verdict): %d/%d' % (len(covered), len(WRITE_FNS)))
    missing = sorted(set(WRITE_FNS) - covered)
    say('never exercised: %s' % (', '.join(missing) if missing else '(none)'))
    if failed_fns:
        say('functions with a FAIL verdict: %s' % ', '.join(sorted(failed_fns)))

    say('')
    say('--- FAIL detail (each one is either a harness bug or a product bug -- say which)')
    any_fail = False
    for r in results:
        for fn, lines in sorted((r.get('verdicts') or {}).items()):
            for ln in lines:
                if ': FAIL(' in ln:
                    any_fail = True
                    say('    %-26s %s' % (r['tag'][:26], ln))
    if not any_fail:
        say('    (none)')

    say('')
    say('--- SKIPs, with the precondition that was missing')
    if not skips:
        say('    (none)')
    for (tag, fn), reason in sorted(skips.items()):
        say('    %-26s %-24s %s' % (tag[:26], fn, reason))

    say('')
    say('--- negative controls (one per write group)')
    groups_seen = {}
    for tag, group, fn, code, good in neg_results:
        groups_seen.setdefault(group, []).append(good)
    for g in sorted(groups_seen):
        oks = sum(1 for x in groups_seen[g] if x)
        say('    %-20s %d/%d refused with a declared code' % (g, oks, len(groups_seen[g])))
    badneg = [(t, f, c) for (t, _g, f, c, good) in neg_results if not good]
    say('    bad negatives: %s' % (badneg if badneg else '(none)'))

    # ---- explicit contract-vs-implementation probes -----------------------------------------
    say('')
    say('--- contract vs implementation')
    nre = [r['tag'] for r in results
           if (r.get('listTables') or (True,))[0] is False and (r.get('listTables') or (0, ''))[1] == 'E_INTERNAL']
    say('    list_tables  (manifest: needsHandle=false, no `handle` param) -> '
        + ('E_INTERNAL NullReferenceException at ListTables on %d/%d files' % (len(nre), len(results))
           if nre else 'ok'))
    nrc = [r['tag'] for r in results
           if (r.get('listColumns') or (True,))[0] is False and (r.get('listColumns') or (0, ''))[1] == 'E_INTERNAL']
    say('    list_columns (manifest: needsHandle=false, no `handle` param) -> '
        + ('E_INTERNAL NullReferenceException at ListColumns on %d/%d files' % (len(nrc), len(results))
           if nrc else 'ok'))
    if nre or nrc:
        say('      both bodies dereference the session (`s.Tzp`) that NeedsHandle=false leaves null,')
        say('      and Manifest.Check refuses a `handle` argument because it is not in their Params --')
        say('      so neither function is reachable at all over the JSON surface.')
    else:
        say('      (fixed: both now read `s == null ? null : s.Tzp`, the pattern PageTab.BaseData')
        say('       already used. This gate is what found them unreachable -- 4/4 files E_INTERNAL.)')

    dbp = [r for r in results if r.get('dbItems')]
    hit = [r for r in dbp if (r['dbItems'][2] or 0) > 0]
    say('    set_items on a column-backed element (fieldType COLUMN_LIKE/TABLE_COLUMN):')
    if hit:
        for r in hit:
            nm, _ok, rtstale, rsample = r['dbItems']
            say('      %-26s %-16s RoundTrip(reload) stale=%s' % (r['tag'][:26], nm, rtstale))
            if rsample:
                say('          ' + rsample[:140])
        say('      the .4fd splice is correct -- a designer RELOAD repopulates `items` from the')
        say('      column, so the write does not survive. Nothing in-session can see it: `verify`')
        say('      compares the live model against the PRISTINE file on disk, not against a reload.')
    else:
        say('      not observed on this sample (%d files had a column-backed ComboBox/RadioGroup)'
            % len(dbp))

    say('')
    say('==== W3-FNS GATE: %s  (%d pass, %d fail) ====' % ('PASS' if g3.ok_all else 'FAIL',
                                                          g3.npass, g3.nfail))

    if not keep:
        for idx in range(len(files)):
            for k in ('pre', 'mid', 'post', '_dbprobe'):
                p = os.path.join(os.path.dirname(files[idx]), '_ai_w3fns_%d_%s.tzs' % (idx, k))
                if os.path.exists(p):
                    os.remove(p)

    try:
        with open(REPORT, 'w', encoding='utf-8') as fh:
            fh.write('\n'.join(report_lines) + '\n')
        print('\nreport: %s' % REPORT)
    except OSError as e:
        print('could not write report: %s' % e)
    return 0 if g3.ok_all else 1


if __name__ == '__main__':
    try:
        sys.stdout.reconfigure(encoding='utf-8', errors='replace')
    except Exception:
        pass
    sys.exit(main())
