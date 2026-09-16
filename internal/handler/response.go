package handler

import (
	"encoding/json"
	"net/http"
)

// respBody 统一响应体
type respBody struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// writeJSON 输出 JSON
func writeJSON(w http.ResponseWriter, status int, body respBody) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// ok 成功响应
func ok(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, respBody{Code: 0, Message: "success", Data: data})
}

// okPage 分页成功响应
func okPage(w http.ResponseWriter, list interface{}, total int64, page, pageSize int) {
	writeJSON(w, http.StatusOK, respBody{Code: 0, Message: "success", Data: map[string]interface{}{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}})
}

// fail 失败响应
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, respBody{Code: status, Message: msg})
}
