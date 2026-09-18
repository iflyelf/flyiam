import request from '@/utils/request'

// 获取登录公开配置（登录地址等）
export function getAuthConfig() {
  return request.get('/api/auth/config')
}

// 获取当前登录用户信息
// silent=true 时不触发全局 401 跳转（用于路由守卫的登录态探测）
export function getUserInfo(silent = false) {
  return request.get('/api/auth/userinfo', silent ? { skipAuthRedirect: true } : undefined)
}

// 退出登录（清除服务端 HttpOnly Cookie）
export function logout() {
  return request.post('/api/auth/logout')
}
