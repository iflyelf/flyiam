package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// NullTime 可空时间类型：可正确扫描 NULL，并序列化为 null 或 RFC3339 字符串
type NullTime struct {
	Time  time.Time
	Valid bool
}

// Scan 实现 sql.Scanner
func (n *NullTime) Scan(value interface{}) error {
	if value == nil {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case time.Time:
		n.Time = v
		return nil
	case []byte:
		t, err := parseTime(string(v))
		if err != nil {
			return err
		}
		n.Time = t
		return nil
	case string:
		t, err := parseTime(v)
		if err != nil {
			return err
		}
		n.Time = t
		return nil
	default:
		return fmt.Errorf("无法转换 %T 为时间", value)
	}
}

// Value 实现 driver.Valuer
func (n NullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

// MarshalJSON 输出 null 或 RFC3339
func (n NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Time)
}

// UnmarshalJSON 接受 null、空字符串或 RFC3339 时间字符串
func (n *NullTime) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" || s == `""` || s == "" {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	// 去掉可能存在的引号
	s = strings.Trim(s, `"`)
	t, err := parseTime(s)
	if err != nil {
		return err
	}
	n.Time, n.Valid = t, true
	return nil
}

// parseTime 兼容多种时间格式
func parseTime(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("不支持的时间格式: %s", s)
}
