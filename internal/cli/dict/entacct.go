package dict

// 客户端直连路径的企业账号解析(ENT → gzou_t → 账号)。
//
// 与 tt debug 同一套规则、同一张 gzou_t 表、同一个快照文件(internal/entdir),
// 区别只在传输方式:debug 在 T100 服务器侧跑 sqlplus/ksql,这里从**客户端直连**库。
// 两条路径必须对一个环境给出同一个账号 —— 否则同一张字典表会被两个 schema 读到,
// 而结果里没有任何信号(那正是"查错了地方被当成没有数据"的根因)。
//
// 降级原则:企业目录查不动**不该让"查一条字典"整个失败**。任何一步出错都退回
// 客户端直连惯例(账号列表首项),但降级原因会作为 note 带进输出信封,不静默。

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/dict/live"
	"tt/internal/entdir"
)

// entLookupTimeout 解析企业账号那一次额外连接的上限(见 fetchEntMappings)。
const entLookupTimeout = 20 * time.Second

// hookEntAccount 解析本次该用哪个账号,并写回连接的 Account 槽位(空 = 用列表首项)。
// 返回的 notes 会被打进输出信封的环境头。
//
// entRaw 是**用户显式给的企业编号**(tt dict --ent <n> / 会话值),不再来自环境配置:
// 环境级的 topent 已随"移除环境级企业"删除(见 config.NamedSsh 的注释)。
// 没给就是没给 —— 退回客户端直连惯例(账号列表首项),并把这件事写进 notes,
// 而不是替用户猜一个企业(猜错的表现是"查不到数据",与"数据不存在"无法区分)。
func hookEntAccount(cc *dbconfig.Connection, ref *config.EnvRef, dataDir, entRaw string) []string {
	acct, source, ent, notes := resolveEntAccount(cc, ref, dataDir, entRaw)
	if acct != "" {
		cc.Account = acct
	}
	if srcTarget != nil {
		srcTarget.WithAccount(acct, source, ent)
	}
	return notes
}

// resolveEntAccount 按**调用方显式给出的企业编号**解析账号。
//
// 返回的 source 为 dbconfig.AcctFromEnt 时 account 才是解析结果;其余情况 account 为空,
// 调用方按客户端直连惯例取账号列表首项。
func resolveEntAccount(cc *dbconfig.Connection, ref *config.EnvRef, dataDir, entRaw string) (account, source string, ent int, notes []string) {
	if entRaw == "" {
		return "", dbconfig.AcctFromPrimary, 0, nil // 没给企业编号:按惯例用列表首项
	}
	ent, err := strconv.Atoi(strings.TrimSpace(entRaw))
	if err != nil || ent <= 0 {
		return "", dbconfig.AcctFromPrimary, 0,
			[]string{fmt.Sprintf("企业编号不是正整数(%s),无法据此解析账号,已按账号列表首项连接", entRaw)}
	}

	// 配置里的 ENT 清单优先：那是人显式同步过的那一份（设置页「ENT」页签 /
	// tt env ents --refresh），存在就**不联网**。环境属于配置，账号解析也不该
	// 因为 VPN 没开而失败；库里的 gzou_t 只在同步那一刻才用。
	if len(ref.Env.Ents) > 0 {
		for _, e := range ref.Env.Ents {
			if e.Ent != ent {
				continue
			}
			acct := strings.TrimSpace(e.Account)
			if acct == "" || acct == "-" {
				return "", dbconfig.AcctFromPrimary, ent,
					[]string{fmt.Sprintf("企业 %d 在配置的 ENT 清单里没有可用账号（该环境已同步 ENT；tt env ents %s 可看清单）", ent, ref.Name)}
			}
			// 账号清单里没收录时，用同步时抄下来的口令（仍按“账号清单优先”的规矩）。
			if e.Password != "" && !inAccountList(cc, acct) {
				cc.Accounts = append(cc.Accounts, dbconfig.DBAcct{Account: acct, Password: e.Password})
			}
			return acct, dbconfig.AcctFromEntConfig, ent, nil
		}
		return "", dbconfig.AcctFromPrimary, ent,
			[]string{fmt.Sprintf("企业 %d 不在配置的 ENT 清单里（该环境已同步 ENT；tt env ents %s 可看清单）", ent, ref.Name)}
	}

	fp := entdir.Fingerprint(ref.Env.Host, ref.Env.User, ref.Env.Zone,
		entdir.DBIdent(cc.Type, cc.Host, cc.Port, cc.Svc()))
	path := entdir.Path(dataDir, entdir.EnvSeg(ref.Name, ref.Env.Host, ref.Env.Zone))

	maps := snapshotMappings(path, fp)
	if maps == nil {
		// 快照没有(冷启动/换环境/过期且从未现查过):直连查一次 gzou_t 并落盘。
		// 这一次查询固定用系统账号 ds(见 dbconfig.SystemAccount) —— 不是账号清单首项。
		m, err := fetchEntMappings(cc)
		if err != nil {
			return "", dbconfig.AcctFromPrimary, ent,
				[]string{"企业账号解析失败,已按账号列表首项连接: " + err.Error()}
		}
		maps = m
		if err := entdir.Write(path, &entdir.Snapshot{
			Version: entdir.Version, Env: ref.Name, Fingerprint: fp, Zone: ref.Env.Zone,
			Dialect: cc.Type, Target: cc.Svc(), Mappings: maps, FetchedAt: time.Now(),
		}); err != nil {
			// 快照只是给下次的便利,写不出去不影响本次结果
			notes = append(notes, "企业目录快照未能落盘(下次仍要现查): "+err.Error())
		}
	}

	acct, err := entAccountOf(maps, ent)
	if err != nil {
		return "", dbconfig.AcctFromPrimary, ent, append(notes, err.Error())
	}
	if !inAccountList(cc, acct) {
		notes = append(notes, "账号 "+acct+" 不在环境 %q 的账号清单里,密码按 T100 惯例取 账号=密码;若连不上,把它加进账号清单")
	}
	return acct, dbconfig.AcctFromEnt, ent, notes
}

// snapshotMappings 读快照里的映射;不可用时返回 nil(交给调用方现查)。
// **不判过期**:过期快照仍是最好的线索,现查失败时它就是唯一答案(与 debug 侧同一条规则)。
func snapshotMappings(path, fingerprint string) []entdir.Mapping {
	snap := entdir.Read(path, fingerprint)
	if snap == nil {
		return nil
	}
	return snap.Mappings
}

// fetchEntMappings 客户端直连连一次,读 gzou_t 的启用企业 → 账号映射。
//
// 与服务器侧那条(dbAllMappings)的区别:它拼字符串,要 nvl/nullif 兜空列;
// 这里拿的是两列原始值,空值在 Go 里判 —— 于是这条 SQL 是纯标准 SELECT,
// 金仓与 Oracle 一个字都不用改。
func fetchEntMappings(cc *dbconfig.Connection) ([]entdir.Mapping, error) {
	// 查 gzou_t 固定用系统账号 **ds**（与 tt debug 服务器侧那条完全相同）:
	// gzou_t 是跨企业共享的系统表，归属由 T100 安装惯例决定；账号清单首项是
	// “人把哪个账号写在了前面”，两者恰好相同不等于同一条规则（见 dbconfig.SystemAccount）。
	lookup := *cc
	lookup.Account = dbconfig.SystemAccount
	// 加超时:这一步发生在"主连接之前",环境连不上时它就是**第二次**超时等待。
	// 上限压在 20s,宁可退回账号列表首项,也不要让人对着一个卡住的命令等两遍。
	ctx, cancel := context.WithTimeout(context.Background(), entLookupTimeout)
	defer cancel()

	l, err := live.Open(ctx, lookup)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	_, rows, err := l.Connector().Query(ctx,
		`SELECT gzou001, gzou003 FROM gzou_t WHERE gzoustus='Y' ORDER BY gzou001`)
	if err != nil {
		return nil, fmt.Errorf("查询 gzou_t 失败: %w", err)
	}
	maps := make([]entdir.Mapping, 0, len(rows))
	for _, r := range rows {
		acct, ok := at(r, 1)
		if !ok {
			continue
		}
		n, ok := at(r, 0)
		if !ok {
			continue
		}
		e, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil || e <= 0 {
			continue
		}
		acct = strings.TrimSpace(acct)
		if acct == "" {
			acct = "-" // 归一:与服务器侧 parseEntLine 同一套"没配"表示
		}
		maps = append(maps, entdir.Mapping{Ent: e, Account: acct, Placeholder: acct == "-"})
	}
	if len(maps) == 0 {
		return nil, errors.New("gzou_t 里没有启用的企业记录")
	}
	return maps, nil
}

// entAccountOf 取某企业的账号;没配账号时给出与 tt debug sql 完全一致的措辞
// (同一件事两条命令说两样话,最容易被当成两个问题)。
func entAccountOf(maps []entdir.Mapping, ent int) (string, error) {
	for _, m := range maps {
		if m.Ent != ent {
			continue
		}
		if a := strings.TrimSpace(m.Account); a != "" && a != "-" {
			return a, nil
		}
		return "", fmt.Errorf("企业 %d 在 gzou_t 里没有配可用账号(该企业的业务数据无法定位到 schema)", ent)
	}
	return "", fmt.Errorf("企业 %d 不在 gzou_t 的启用清单里", ent)
}

// inAccountList 该账号是否在环境的账号清单里。
func inAccountList(cc *dbconfig.Connection, acct string) bool {
	for _, a := range cc.Accounts {
		if a.Account == acct {
			return true
		}
	}
	return false
}

// at 取行内第 i 列(缺列返回 ok=false)。
func at(row []string, i int) (string, bool) {
	if i < 0 || i >= len(row) {
		return "", false
	}
	return row[i], true
}
