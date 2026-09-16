package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray 用于扫描/写入 PostgreSQL TEXT[] 字段
type StringArray []string

// Scan 实现 sql.Scanner
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = StringArray{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*a = parsePgArray(string(v))
		return nil
	case string:
		*a = parsePgArray(v)
		return nil
	default:
		return fmt.Errorf("无法转换 %T 为字符串数组", value)
	}
}

// Value 实现 driver.Valuer
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return encodePgArray(a), nil
}

// parsePgArray 解析 PostgreSQL 数组字面量，如 {read,write}
func parsePgArray(s string) StringArray {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return StringArray{}
	}
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if s == "" {
		return StringArray{}
	}

	var out StringArray
	var buf strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ',' && !inQuote:
			out = append(out, buf.String())
			buf.Reset()
		case c == '\\' && inQuote && i+1 < len(s):
			i++
			buf.WriteByte(s[i])
		default:
			buf.WriteByte(c)
		}
	}
	out = append(out, buf.String())
	return out
}

// encodePgArray 编码为 PostgreSQL 数组字面量
func encodePgArray(a StringArray) string {
	if len(a) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(a))
	for _, item := range a {
		escaped := strings.ReplaceAll(item, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		parts = append(parts, `"`+escaped+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}
