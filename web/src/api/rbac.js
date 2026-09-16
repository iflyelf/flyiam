import request from '@/utils/request'

// ============================ 角色 ============================
export function listRoles(params) {
  return request.get('/api/roles', { params })
}

export function createRole(data) {
  return request.post('/api/roles', data)
}

export function updateRole(id, data) {
  return request.put(`/api/roles/${id}`, data)
}

export function deleteRole(id) {
  return request.delete(`/api/roles/${id}`)
}

// ============================ 团队 ============================
export function listTeams(params) {
  return request.get('/api/teams', { params })
}

export function createTeam(data) {
  return request.post('/api/teams', data)
}

export function updateTeam(id, data) {
  return request.put(`/api/teams/${id}`, data)
}

export function deleteTeam(id) {
  return request.delete(`/api/teams/${id}`)
}

export function listTeamMembers(id) {
  return request.get(`/api/teams/${id}/members`)
}

export function addTeamMember(id, data) {
  return request.post(`/api/teams/${id}/members`, data)
}

export function removeTeamMember(id, username) {
  return request.delete(`/api/teams/${id}/members/${encodeURIComponent(username)}`)
}

export function listTeamRoles(id) {
  return request.get(`/api/teams/${id}/roles`)
}

export function setTeamRoles(id, roleIds) {
  return request.post(`/api/teams/${id}/roles`, { roleIds })
}

// ============================ 权限清单 ============================
export function listPermissions() {
  return request.get('/api/permissions')
}
