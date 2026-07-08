package auth

import "strings"

// MatchScope 检查 scope 是否匹配模型
// scope 格式示例: "gpt-4,gpt-4-turbo" 或 "*"（匹配所有）
// model 为 "*" 时表示"不限制模型"，同样放行
func MatchScope(scopes, model string) bool {
	if scopes == "*" || model == "*" {
		return true
	}

	scopeList := strings.Split(scopes, ",")
	for _, s := range scopeList {
		s = strings.TrimSpace(s)
		if s == model {
			return true
		}
	}
	return false
}
