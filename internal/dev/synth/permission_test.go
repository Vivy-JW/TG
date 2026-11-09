package synth

import (
	"testing"

	"tt/internal/dev/model"
)

// envC / envS 是两套典型的包级上下文（对应真实语料 root 属性）。
func envC() model.EnvContext {
	return model.EnvContext{Env: "c", Topind: "sd", LoginUser: "tiptop", ProgType: "M", IsStandard: true}
}
func envS() model.EnvContext {
	return model.EnvContext{Env: "s", Topind: "sd", LoginUser: "topstd", ProgType: "M", IsStandard: true}
}

// TestResolvePointTruthTable 逐条钉住 G1–G7（AddPointModel.cs:691-747）。
func TestResolvePointTruthTable(t *testing.T) {
	base := map[string]string{"src": "s", "status": "", "cite_std": "N", "readonly": "", "edit": ""}
	mk := func(kv map[string]string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range kv {
			m[k] = v
		}
		return m
	}
	name := "function.f"

	cases := []struct {
		desc     string
		name     string
		meta     map[string]string
		env      model.EnvContext
		selfDef  bool
		parseOK  bool
		wantOK   bool
		wantDeny string
	}{
		{"普通客制环境 + src=s → 可编辑", name, mk(nil), envC(), true, true, true, model.DenyNone},
		{"G1 独立功能程序行业别不匹配 → 拒", name,
			mk(map[string]string{"ind_fun": "xy"}), func() model.EnvContext {
				e := envC()
				e.IsIndFun = true
				return e
			}(), true, true, false, model.DenyIndFunMismatch},
		{"G1 例外 global.memo_industry → 放行", "global.memo_industry",
			mk(map[string]string{"ind_fun": "xy"}), func() model.EnvContext {
				e := envC()
				e.IsIndFun = true
				return e
			}(), false, true, true, model.DenyNone},
		{"G2 标准环境 + sd 行业别 + industry memo → 拒", "global.memo_industry",
			mk(nil), envS(), false, true, false, model.DenyIndustryMemo},
		{"G3 非标准件 + cite_std=Y → 拒", name,
			mk(map[string]string{"cite_std": "Y"}), func() model.EnvContext {
				e := envC()
				e.IsStandard = false
				return e
			}(), true, true, false, model.DenyCiteStd},
		{"G4 readonly=Y → 拒", name, mk(map[string]string{"readonly": "Y"}), envC(), true, true, false, model.DenyReadonlyAttr},
		{"G5a edit=c 且 env=s → 拒", name, mk(map[string]string{"edit": "c"}), envS(), true, true, false, model.DenyEnvEditEnv},
		{"G5a edit=c 且 env=c 且 topstd → 拒", name,
			mk(map[string]string{"edit": "c"}), func() model.EnvContext {
				e := envC()
				e.IsTopstdMode = true
				return e
			}(), true, true, false, model.DenyEnvEditEnv},
		{"G5b edit=s 且 env=c 且 src!=c → 拒", name,
			mk(map[string]string{"edit": "s", "src": "s"}), envC(), true, true, false, model.DenyEnvEditEnv},
		{"G6 topstd + src=c + status=u → 拒", name,
			mk(map[string]string{"src": "c", "status": "u"}), func() model.EnvContext {
				e := envC()
				e.IsTopstdMode = true
				return e
			}(), true, true, false, model.DenyTopstdMode},
		{"G6 topstd + src=c + status=c → 放行", name,
			mk(map[string]string{"src": "c", "status": "c"}), func() model.EnvContext {
				e := envC()
				e.IsTopstdMode = true
				return e
			}(), true, true, true, model.DenyNone},
		{"G7 login=topstd + src=c + status=u → 拒", name,
			mk(map[string]string{"src": "c", "status": "u"}), envS(), true, true, false, model.DenyLoginTopstd},
		{"结构行解析不出（自订定义点）→ 拒", name, mk(nil), envC(), true, false, false, model.DenyStructureUnparsable},
		{"结构行解析不出（裸名点不受影响）→ 放行", "global.memo", mk(nil), envC(), false, false, true, model.DenyNone},
	}
	for _, c := range cases {
		ok, deny := ResolvePoint(c.name, c.meta, c.env, c.selfDef, c.parseOK)
		if ok != c.wantOK || deny != c.wantDeny {
			t.Errorf("%s：得到 (ok=%v, deny=%q)，想要 (ok=%v, deny=%q)", c.desc, ok, deny, c.wantOK, c.wantDeny)
		}
	}
}

// TestResolveSectionTruthTable 逐条钉住 S1–S5（SectionModel.cs:101-133）+ tdev 政策层。
func TestResolveSectionTruthTable(t *testing.T) {
	prog := "adzi999"
	base := map[string]string{"src": "s", "status": ""}
	mk := func(kv map[string]string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range kv {
			m[k] = v
		}
		return m
	}
	unlock := func(e model.EnvContext) model.EnvContext { e.SectionState = model.SectionUnlocked; return e }

	cases := []struct {
		desc     string
		name     string
		readonly bool
		meta     map[string]string
		env      model.EnvContext
		wantOK   bool
		wantDeny string
	}{
		{"Locked（框架未解开）→ section-locked", prog + ".main", false, mk(nil), envC(), false, model.DenySecLocked},
		{"Unlocked + src=s → 可编辑", prog + ".main", false, mk(nil), unlock(envC()), true, model.DenyNone},
		{"锚点区段永不写（即使开了 SEC）", prog + ".other_function", false, mk(nil), unlock(envC()), false, model.DenySectionAnchor},
		{"S2 TGL readonly=Y → 拒", prog + ".main", true, mk(nil), unlock(envC()), false, model.DenySectionReadonly},
		{"S3 readonly 属性=Y → 拒", prog + ".main", false, mk(map[string]string{"readonly": "Y"}), unlock(envC()), false, model.DenyReadonlyAttr},
		{"S1 type=G + section_flag=Y → 可编辑（非锚点）", prog + ".main", true, mk(nil), func() model.EnvContext {
			e := unlock(envC())
			e.ProgType = "G"
			e.SectionFlag = true
			return e
		}(), true, model.DenyNone},
		{"S1 type=G 下 other_report 仍只读", prog + ".other_report", true, mk(nil), func() model.EnvContext {
			e := unlock(envC())
			e.ProgType = "G"
			e.SectionFlag = true
			return e
		}(), false, model.DenySectionAnchor},
		{"S4 topstd + src=c → 拒", prog + ".main", false, mk(map[string]string{"src": "c"}), func() model.EnvContext {
			e := unlock(envC())
			e.IsTopstdMode = true
			return e
		}(), false, model.DenyTopstdMode},
		{"S5 login=topstd + src=c → 拒", prog + ".main", false, mk(map[string]string{"src": "c"}), unlock(envS()), false, model.DenyLoginTopstd},
	}
	for _, c := range cases {
		ok, deny := ResolveSection(c.name, c.readonly, c.meta, c.env, prog)
		if ok != c.wantOK || deny != c.wantDeny {
			t.Errorf("%s：得到 (ok=%v, deny=%q)，想要 (ok=%v, deny=%q)", c.desc, ok, deny, c.wantOK, c.wantDeny)
		}
	}
}
