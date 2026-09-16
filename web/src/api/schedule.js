import request from '@/utils/request'

// 获取定时任务配置
export function getSchedule() {
  return request.get('/api/schedule')
}

// 更新定时任务配置
export function updateSchedule(data) {
  return request.put('/api/schedule', data)
}
