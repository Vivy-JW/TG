package config

import (
	"encoding/json"
	"strings"
)

// 配置里的口令打码。
//
// 用**键名**判断而不是结构体标签:未知节的未知键也要一起打码
// (设置页可以写进任意键,靠标签一定会漏)。
//
// 为什么这件事要抽到这里:`tt config show` 早就打码了,但 `tt config get`、
// `tt dict db list --json`、`GET /api/hosts` 三处仍然原样吐明文 —— 同一份配置
// 三处三种口径。规则只有一份实现,各处调它。

// IsSecretKey 键名是否表示口令。
func IsSecretKey(k string) bool {
	switch strings.ToLower(k) {
	case "password", "passwd", "pwd":
		return true
	}
	return false
}

// SecretPath 点分路径是否**指向**口令(任一段是口令键即为真)。
// 用于 `tt config get hosts.sshs.0.db.accounts.0.password` 这种直接点名取值。
func SecretPath(path string) bool {
	for _, seg := range strings.Split(path, ".") {
		if IsSecretKey(seg) {
			return true
		}
	}
	return false
}

// RedactSecrets 深度复制一份配置树,把口令字段替换成占位符。
// 只认 map/slice;标量原样返回(要点名取值时用 SecretPath 判断)。
func RedactSecrets(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if IsSecretKey(k) {
				if s, ok := val.(string); ok && s != "" {
					out[k] = "***"
					continue
				}
			}
			out[k] = RedactSecrets(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = RedactSecrets(e)
		}
		return out
	default:
		return v
	}
}

// RedactValue 把任意值先转成通用的 map/slice 视图再打码 ——
// 给结构体载荷用(如 db list 的行),这样打码规则仍然只有 RedactSecrets 一处。
// 序列化失败时原样返回:打码不该让命令报错。
func RedactValue(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var m any
	if err := json.Unmarshal(b, &m); err != nil {
		return v
	}
	return RedactSecrets(m)
}
