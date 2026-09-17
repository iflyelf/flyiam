import request from '@/utils/request'

// 用户字段定义列表
export function listUserFields() {
  return request.get('/api/user-fields')
}

// 新增字段定义
export function createUserField(data) {
  return request.post('/api/user-fields', data)
}

// 更新字段定义
export function updateUserField(id, data) {
  return request.put(`/api/user-fields/${id}`, data)
}

// 删除字段定义
export function deleteUserField(id) {
  return request.delete(`/api/user-fields/${id}`)
}
