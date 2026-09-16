import request from '@/utils/request'

// 数据源配置列表
export function listDataSources() {
  return request.get('/api/datasources')
}

// 创建数据源
export function createDataSource(data) {
  return request.post('/api/datasources', data)
}

// 更新数据源
export function updateDataSource(id, data) {
  return request.put(`/api/datasources/${id}`, data)
}

// 删除数据源
export function deleteDataSource(id) {
  return request.delete(`/api/datasources/${id}`)
}

// 测试已保存的数据源
export function testDataSource(id) {
  return request.post(`/api/datasources/${id}/test`)
}

// 测试未保存的数据源配置
export function testDataSourceConfig(data) {
  return request.post('/api/datasources/test', data)
}
