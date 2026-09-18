package model

import "time"

// Setting 应用设置（页面可配置，DB 优先 / env 兜底）
type Setting struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}
