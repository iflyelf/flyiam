import request from '@/utils/request'

// ============================ 受保护用户 ============================
export function listProtectedUsers() {
  return request.get('/api/protected-users')
}

export function addProtectedUser(data) {
  return request.post('/api/protected-users', data)
}

export function deleteProtectedUser(account) {
  return request.delete(`/api/protected-users/${encodeURIComponent(account)}`)
}

// ============================ 应用管理 ============================
export function listApplications() {
  return request.get('/api/casdoor/apps')
}

export function getApplication(name) {
  return request.get(`/api/casdoor/apps/${encodeURIComponent(name)}`)
}

export function createApplication(data) {
  return request.post('/api/casdoor/apps', data)
}

export function updateApplication(name, data) {
  return request.put(`/api/casdoor/apps/${encodeURIComponent(name)}`, data)
}

export function deleteApplication(name) {
  return request.delete(`/api/casdoor/apps/${encodeURIComponent(name)}`)
}

// ============================ 组织管理 ============================
export function listOrganizations() {
  return request.get('/api/casdoor/orgs')
}

export function createOrganization(data) {
  return request.post('/api/casdoor/orgs', data)
}

export function updateOrganization(name, data) {
  return request.put(`/api/casdoor/orgs/${encodeURIComponent(name)}`, data)
}

export function deleteOrganization(name) {
  return request.delete(`/api/casdoor/orgs/${encodeURIComponent(name)}`)
}
