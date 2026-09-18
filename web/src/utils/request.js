import axios from 'axios'
import { ElMessage } from 'element-plus'

// 同源请求（生产环境前后端一体，开发环境由 Vite 代理 /api）
// 登录凭证由后端 HttpOnly Cookie 承载，浏览器自动携带，前端不保存 token。
const request = axios.create({
  baseURL: '',
  timeout: 120000,
  withCredentials: true
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
      localStorage.removeItem('flyiam_user')
      // 登录态探测请求（skipAuthRedirect）不触发跳转，交由路由守卫处理
      const skip = err.config?.skipAuthRedirect
      if (!skip && window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(err)
  }
)

export default request
