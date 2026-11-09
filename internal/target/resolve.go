package target

// 落位：把一次寻址请求在会话池视图上落成**唯一一条会话**（契约 2.2 / 2.4 / C2.7–C2.12 /
// C3.1–C3.3 / C8.1 / Q23）。全部是纯函数：不发请求、不写盘。

// NameResolution 是 `--name X` 的三种结局（Q23 / C1.6）。
type NameResolution int

const (
	// NameFree：会话池里没有这条别名 → 起新会话（序号照 Q16 / Q24 由**服务端**自增）。
	NameFree NameResolution = iota
	// NameReuse：别名已存在且那条会话**空闲（idle）** → **复用**它（Q23 的幂等键）。
	NameReuse
	// NameBusy：别名已存在且那条会话**正在跑 / 已停站**（以及连接中/已断开）→ 报 `SESSION_TARGET_BUSY`。
	//
	// 只有 `idle` 才复用：Q23 的原话是"空闲（idle）→ 复用它"，而"半启动"（loading）
	// 与"已断气"（exit）都不是能干活的状态；把它们当可复用会让第二条命令落进一条起不来的会话。
	NameBusy
)

// Resolve 把一次寻址请求落到**唯一**一条会话上。
//
// 成功返回 (会话, nil)；失败返回 (零值, 失败信封的数据)。
// 判定顺序（写死，因为它决定"同时踩两条规矩时报哪一条"）：
//
//	① 文法与标志冲突（F9–F13）：**发请求之前**判定（C2.8）；
//	② 环境存在性（F6 / C2.1）：不猜、不回落；
//	③ 池空（F2）：池里一条都没有时，"你要的那条"与"你要的那个作业"都不存在；
//	④ 取候选集（`--env` 是环境的权威，C3.1）；
//	⑤ 按选择器形态落位（F1 / F3–F8、C2.12）；
//	⑥ 世代校验（F15，先于企业核对：句柄过期时它的内容已经没有意义）；
//	⑦ 企业核对（F14 / C3.3-3）。
func Resolve(pool Pool, req Request) (Session, *Failure) {
	sel := req.Selector

	// ⓪ 写法 ⑧ 与别名 ③ 同形（C2.4 的分流只看 `/` 与 `#`）：拿到环境清单之后先分清它们。
	//    优先级是"环境名赢"，理由见 promoteEnvName。
	if sel != nil {
		promoted := promoteEnvName(*sel, pool.Environments)
		sel = &promoted
		req.Selector = sel
	}

	// ① 文法与标志冲突：F12（环境矛盾）先于 F13（企业矛盾）—— C2.9 明写"环境不同就先撞环境那条"。
	if sel != nil {
		if sel.Env != "" && req.Env != "" && sel.Env != req.Env {
			return Session{}, failureEnvConflict(sel.Env, req.Env)
		}
		if sel.Kind == SelectorEnvEnt && req.Ent != "" && sel.Ent != req.Ent {
			return Session{}, failureEntConflict(sel.Ent, req.Ent)
		}
	}

	// ② 环境存在性：选择器里的环境、或 `--env` 给的那个，都必须在配置里（F6 / C2.1）。
	env := effectiveEnv(req)
	if env != "" && !containsStr(pool.Environments, env) {
		return Session{}, failureEnvUnknown(env, pool.Environments)
	}

	// ③ 池空 = 没有会话（F2）：先于一切按选择器的判定。
	if len(pool.Sessions) == 0 {
		return Session{}, failureNoSession()
	}

	// ④ 候选集：`--env` 是**环境**的权威（C3.1），给了就把池收窄到它。
	cand := pool.Sessions
	if env != "" {
		cand = filterByEnv(pool.Sessions, env)
	}

	if sel == nil {
		return pickUnique(cand, nil, req, env)
	}

	switch sel.Kind {
	case SelectorAlias:
		hits := filterByAlias(cand, sel.Alias)
		if len(hits) == 0 {
			// F5：候选块**只列有名字的那些**（2.4 的 F5 行）。
			return Session{}, failureAliasNotFound(sel.Alias, filterNamed(cand))
		}
		if len(hits) > 1 {
			// 别名查重是不区分大小写的（C2.5），但按 C2.5 末句查找按原样比 ——
			// 真出现两条同名时不能替调用方挑一条。
			return Session{}, failureAmbiguous(sel.Raw, hits)
		}
		return finish(hits[0], req)

	case SelectorEnv:
		return pickUnique(cand, sel, req, sel.Env)

	case SelectorEnvJob:
		hits := filterByProg(cand, sel.Prog)
		if len(hits) == 0 {
			return Session{}, failureJobNotFound(sel.Env, sel.Prog, cand) // F7
		}
		if len(hits) > 1 {
			return Session{}, failureAmbiguous(sel.Raw, hits) // F3
		}
		return finish(hits[0], req)

	case SelectorEnvEnt:
		hits := filterByEnt(cand, sel.Ent)
		if len(hits) == 0 {
			return Session{}, failureEntNotFound(sel.Env, sel.Ent, cand)
		}
		if len(hits) > 1 {
			return Session{}, failureAmbiguous(sel.Raw, hits)
		}
		return finish(hits[0], req)

	case SelectorRef:
		hits := filterBySeq(cand, sel.N)
		if len(hits) == 0 {
			// F8：`<M>` 是**同一环境下**的会话条数（C2.11），候选块列的就是这 M 条。
			return Session{}, failureSeqOutOfRange(sel.Env, len(cand), sel.N, cand)
		}
		return finish(hits[0], req)

	case SelectorEnvJobSeq:
		hits := filterBySeq(cand, sel.N)
		if len(hits) == 0 {
			return Session{}, failureSeqOutOfRange(sel.Env+SelectorSep+sel.Prog, len(cand), sel.N, cand)
		}
		// C2.12：先按 `环境#序号` 定位候选，再用作业段核对；核对不过就是
		// "你要的那个作业没有这条会话" → 按 F7 报，**不静默落**。
		//
		// 作业段只与视图的 `prog` 比：`display` 用的就是它，而 `runProg` 是"实际跑的程序"
		// 快照，按 C2.5c 不参与寻址（快照不进地址）。
		if hits[0].Prog != sel.Prog {
			return Session{}, failureJobNotFound(sel.Env, sel.Prog, cand)
		}
		return finish(hits[0], req)
	}

	// 走不到：SelectorKind 的六种都在上面。留着是为了"新增一种形态忘了接"时有一个响。
	return Session{}, failureSelectorGrammar(sel.Raw)
}

// ResolveName 判 `--name X` 的三种结局（Q23 / C1.6 / C3.7 / C3.7b）。
//
//	会话池里没有这条别名        → (NameFree, 零值, nil)：起新会话；
//	有，且那条会话空闲（idle）   → (NameReuse, 那条, nil)：**复用它**（幂等键）；
//	有，但不能复用（跑/停站…）   → (NameBusy, 那条, SESSION_TARGET_BUSY)：退 3，**不许静默再开一条**。
//
// 别名的查重**不区分大小写**（C2.5：两个只差大小写的同名别名算撞车，F5/F18）。
// 作用域是**整个会话池**：`--session <别名>` 的查找也跨环境，两处必须是同一个范围，
// 否则会出现"起的时候说没撞、找的时候找到两条"。
func ResolveName(pool []Session, name string) (NameResolution, Session, *Failure) {
	for _, s := range pool {
		if s.Name == "" || !SameAlias(s.Name, name) {
			continue
		}
		if s.State == StateIdle {
			return NameReuse, s, nil
		}
		return NameBusy, s, FailureTargetBusy(name, s)
	}
	return NameFree, Session{}, nil
}

// ---------------------------------------------------------------------------
// 内部：候选集收窄与落位
// ---------------------------------------------------------------------------

// pickUnique 处理"按环境找"与"省略目标"（写法 ⑧ 与 ①）：唯一就落（C8.1），多条报歧义。
func pickUnique(cand []Session, sel *Selector, req Request, env string) (Session, *Failure) {
	switch {
	case len(cand) == 1:
		return finish(cand[0], req)
	case len(cand) == 0:
		// 池里有会话，但收窄后一条都不剩（`--env` 指到一个还没有会话的环境）。
		// 报"这个环境下没有会话"而不是"整池空"（C2.7：指名的东西不存在 = SESSION_NOT_FOUND）。
		scope := env
		if sel != nil {
			scope = sel.Env
		}
		return Session{}, failureNoSessionInScope(scope, nil)
	default:
		if sel == nil {
			return Session{}, failureAmbiguousOmitted(cand) // F1
		}
		return Session{}, failureAmbiguous(sel.Raw, cand) // F4
	}
}

// finish 是落位之后的最后两道核对：世代（F15）与企业（F14）。
func finish(s Session, req Request) (Session, *Failure) {
	if req.Epoch != "" && s.Epoch != req.Epoch {
		return Session{}, failureStale(s, req.Epoch) // F15
	}
	if req.Ent != "" && s.Ent != req.Ent {
		return Session{}, failureSessionEntConflict(s.CanonicalRef(), s.Ent, req.Ent) // F14
	}
	return s, nil
}

// effectiveEnv 取本次调用真正生效的环境名：选择器里的环境优先（它是 `--session` 那一层的），
// 否则用 `--env`。两者矛盾时在 `Resolve` 的 ① 就报 F12 了，所以到这里不会有两说。
func effectiveEnv(req Request) string {
	if req.Selector != nil && req.Selector.Env != "" {
		return req.Selector.Env
	}
	return req.Env
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func filterByEnv(in []Session, env string) []Session {
	out := make([]Session, 0, len(in))
	for _, s := range in {
		if s.Env == env {
			out = append(out, s)
		}
	}
	return out
}

func filterByAlias(in []Session, alias string) []Session {
	out := make([]Session, 0, len(in))
	for _, s := range in {
		if s.Name != "" && AliasMatches(s.Name, alias) {
			out = append(out, s)
		}
	}
	return out
}

// filterNamed 返回有名字的那些会话（F5 的候选块：只列有名字的）。
func filterNamed(in []Session) []Session {
	out := make([]Session, 0, len(in))
	for _, s := range in {
		if s.Name != "" {
			out = append(out, s)
		}
	}
	return out
}

func filterByProg(in []Session, prog string) []Session {
	out := make([]Session, 0, len(in))
	for _, s := range in {
		if s.Prog == prog {
			out = append(out, s)
		}
	}
	return out
}

func filterByEnt(in []Session, ent string) []Session {
	out := make([]Session, 0, len(in))
	for _, s := range in {
		if s.Ent == ent {
			out = append(out, s)
		}
	}
	return out
}

func filterBySeq(in []Session, n int) []Session {
	out := make([]Session, 0, 1)
	for _, s := range in {
		if s.N == n {
			out = append(out, s)
		}
	}
	return out
}
