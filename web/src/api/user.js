import request from '@/utils/request'

// 用户列表（服务端分页 + 搜索，数据来自 Casdoor）
export function listUsers(params) {
  return request.get('/api/users', { params })
}

// 用户搜索（跨域账号/姓名，供选择器下拉使用；不分页）
export function searchUsers(keyword = '', limit = 50) {
  return request.get('/api/users/search', { params: { keyword, limit } })
}

// 用户详情
export function getUserDetail(domainAccount) {
  return request.get('/api/user/detail', { params: { domainAccount } })
}

// 用户统计
export function getUserStats() {
  return request.get('/api/user/stats')
}

// 新增用户
export function createUser(data) {
  return request.post('/api/users', data)
}

// 更新用户
export function updateUser(name, data) {
  return request.put(`/api/users/${encodeURIComponent(name)}`, data)
}

// 删除用户
export function deleteUser(name) {
  return request.delete(`/api/users/${encodeURIComponent(name)}`)
}

// 批量删除用户
export function batchDeleteUsers(names) {
  return request.post('/api/users/batch-delete', { names })
}

// 重置用户密码（留空使用系统默认密码）
export function resetPassword(name, password = '') {
  return request.post(`/api/users/${encodeURIComponent(name)}/reset-password`, { password })
}

// 设置用户管理员标记
export function setUserAdmin(name, isAdmin) {
  return request.post(`/api/users/${encodeURIComponent(name)}/admin`, { isAdmin })
}

// 当前用户修改自己的密码
export function changePassword(oldPassword, newPassword) {
  return request.post('/api/auth/change-password', { oldPassword, newPassword })
}
