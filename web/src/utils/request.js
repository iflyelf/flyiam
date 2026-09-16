import axios from 'axios'
import { ElMessage } from 'element-plus'

// 同源请求（生产环境前后端一体，开发环境由 Vite 代理 /api）
const request = axios.create({
  baseURL: '',
  timeout: 120000
})

request.interceptors.request.use((config) => {
  const token = localStorage.getItem('flyiam_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body.code !== 'undefined' && body.code !== 0) {
      ElMessage.error(body.message || '请求失败')
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return body
  },
  (err) => {
    const status = err.response?.status
    const msg = err.response?.data?.message || err.message || '请求失败'
    if (status === 401) {
      localStorage.removeItem('flyiam_token')
      localStorage.removeItem('flyiam_user')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(err)
  }
)

export default request
