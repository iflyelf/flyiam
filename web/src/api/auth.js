import request from '@/utils/request'

// 获取登录公开配置（登录地址等）
export function getAuthConfig() {
  return request.get('/api/auth/config')
}

// 获取当前登录用户信息
export function getUserInfo() {
  return request.get('/api/auth/userinfo')
}
