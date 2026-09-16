// 统一时间格式化：后端返回 UTC 时间字符串，这里按上海时区展示
const TZ = 'Asia/Shanghai'

// formatDateTime 格式化为 YYYY-MM-DD HH:mm:ss
export function formatDateTime(value) {
  if (!value) return '-'
  const d = new Date(value)
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '-'
  const parts = new Intl.DateTimeFormat('zh-CN', {
    timeZone: TZ,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).formatToParts(d)
  const get = (type) => parts.find((p) => p.type === type)?.value || '00'
  return `${get('year')}-${get('month')}-${get('day')} ${get('hour')}:${get('minute')}:${get('second')}`
}

// formatDuration 格式化毫秒耗时
export function formatDuration(ms) {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}
