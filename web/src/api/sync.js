import request from '@/utils/request'

// 从数据源同步
export function syncDataSource() {
  return request.post('/api/sync/datasource')
}

// 同步到 Casdoor
export function syncCasdoor(batchSize = 50) {
  return request.post('/api/sync/casdoor', null, { params: { batchSize } })
}

// 完整同步
export function syncFull(batchSize = 50) {
  return request.post('/api/sync/full', null, { params: { batchSize } })
}

// 同步日志
export function getSyncLogs(params) {
  return request.get('/api/sync/logs', { params })
}

// 同步进度
export function getSyncProgress() {
  return request.get('/api/sync/progress')
}
