import request from '@/utils/request'

// 我的 API 令牌列表
export function listTokens() {
  return request.get('/api/tokens')
}

// 生成令牌（返回明文，仅一次）
export function createToken(data) {
  return request.post('/api/tokens', data)
}

// 吊销令牌
export function revokeToken(id) {
  return request.delete(`/api/tokens/${id}`)
}
