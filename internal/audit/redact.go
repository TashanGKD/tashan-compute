package audit

import "strings"

const redacted = "[REDACTED]"

func Redact(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, nested := range typed {
			if sensitiveKey(key) {
				result[key] = redacted
				continue
			}
			result[key] = Redact(nested)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, nested := range typed {
			result[index] = Redact(nested)
		}
		return result
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(strings.ToLower(key))
	for _, marker := range []string{"password", "token", "authorization", "cookie", "secret", "private_key"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
