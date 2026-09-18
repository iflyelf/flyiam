// API 基地址配置
//
// 同源部署（前端嵌入后端，默认）：留空，使用相对路径。
// 前后端跨域部署：构建时设置 VITE_API_BASE_URL 为后端地址，例如
//   VITE_API_BASE_URL=https://flyiam-api.example.com
// 同时后端需配置 CORS_ALLOWED_ORIGINS 与 AUTH_COOKIE_SAMESITE=none。
export const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

// apiUrl 拼接完整 API 地址
export const apiUrl = (path) => `${API_BASE}${path}`
