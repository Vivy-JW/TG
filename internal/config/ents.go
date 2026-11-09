package config

// ENT 清单的落盘：`tt env ents --refresh` 与设置页「ENT」页签的“从服务器刷新”共用这一处。
//
// 为什么写进 config.json 而不是只留快照：环境属于配置，企业清单是慢变量，
// “能看到可选项”不该依赖 VPN/内网是否通。写进来之后 `--at-list` / `tt env ents`
// 都是纯本地读取；同步是唯一需要连网的动作。
//
// 同时把清单里的账号补进 `db.accounts`（缺的才补，密码优先用已有条目、否则按
// T100 惯例 账号=密码）：这样“账号清单”仍然是口令唯一的编辑入口，
// ents 里的 password 只是同步那一刻的副本（`tt config show` 会打码）。

import (
	"fmt"
	"time"
)

// SyncedEnt 一条待落盘的 ENT（ent + 账号 + 同步时用的口令）。
type SyncedEnt struct {
	Ent      int
	Account  string
	Password string
}

// SaveEnvEnts 把某环境的 ENT 清单写进 config.json（原子写，只动该环境这一项）。
//
// 返回本次**新增**到 db.accounts 的账号名：它们的口令是按惯例填的（账号=密码），
// 调用方应把这件事说出来 —— 否则一次“连接失败”会被误读成“库里没这个号”。
func SaveEnvEnts(path, env string, ents []SyncedEnt, syncedAt time.Time) ([]string, error) {
	var added []string
	err := Edit(path, nil, func(root map[string]any) error {
		hosts, _ := root["hosts"].(map[string]any)
		if hosts == nil {
			return fmt.Errorf("配置里没有 hosts 节")
		}
		list, _ := hosts["sshs"].([]any)
		for _, it := range list {
			m, ok := it.(map[string]any)
			if !ok || stringOf(m["name"]) != env {
				continue
			}

			// 1) 账号清单：缺的补上，已有的以清单里的口令为准（那是人编辑的地方）。
			db, _ := m["db"].(map[string]any)
			accounts, _ := db["accounts"].([]any)
			byAcct := make(map[string]map[string]any, len(accounts))
			for _, a := range accounts {
				am, ok := a.(map[string]any)
				if !ok {
					continue
				}
				byAcct[stringOf(am["account"])] = am
			}
			for i := range ents {
				e := &ents[i]
				if e.Account == "" {
					continue
				}
				if am, ok := byAcct[e.Account]; ok {
					if p := stringOf(am["password"]); p != "" {
						// 口令以账号清单为准；把这份抄进 ents，保证两份一致、可离线展示。
						e.Password = p
					} else if e.Password != "" {
						am["password"] = e.Password
					}
					continue
				}
				if db == nil {
					continue // 该环境没配库：只写清单，不凭空造账号
				}
				p := e.Password
				if p == "" {
					p = e.Account // T100 惯例：未收录的账号，密码=账号
				}
				e.Password = p
				accounts = append(accounts, map[string]any{"account": e.Account, "password": p})
				added = append(added, e.Account)
			}
			if db != nil {
				db["accounts"] = accounts
			}

			// 2) 清单本体 + 同步时间。
			arr := make([]any, 0, len(ents))
			for _, e := range ents {
				obj := map[string]any{"ent": e.Ent, "account": e.Account}
				if e.Password != "" {
					obj["password"] = e.Password
				}
				arr = append(arr, obj)
			}
			m["ents"] = arr
			m["entsSyncedAt"] = syncedAt.Format(time.RFC3339)
			return nil
		}
		return fmt.Errorf("环境 %q 不存在", env)
	})
	if err != nil {
		return nil, err
	}
	return added, nil
}
