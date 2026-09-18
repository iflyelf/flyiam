import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getUserInfo, logout as apiLogout } from '@/api/auth'

// 仅缓存非敏感的用户展示信息；登录凭证由后端 HttpOnly Cookie 承载，
// 前端不接触 token（避免 localStorage 被 XSS 窃取）。
const USER_KEY = 'flyiam_user'

export const useAuthStore = defineStore('auth', () => {
  const userInfo = ref(JSON.parse(localStorage.getItem(USER_KEY) || 'null'))
  // 登录态以服务端探测为准（Cookie 不可被 JS 读取）
  const isAuthenticated = ref(false)
  // 是否已完成一次登录态探测，避免路由守卫重复请求
  const initialized = ref(false)

  // 跳转到 Casdoor 登录（OAuth2 授权码模式，地址由后端动态生成）
  const login = () => {
    window.location.href = '/api/auth/login'
  }

  // 处理 Casdoor 回调：凭证已由后端写入 Cookie，这里只拉取用户信息
  const handleCallback = async () => {
    const res = await getUserInfo()
    userInfo.value = res.data || {}
    localStorage.setItem(USER_KEY, JSON.stringify(userInfo.value))
    isAuthenticated.value = true
    return userInfo.value
  }

  // 刷新当前用户信息（含权限）
  const refreshUserInfo = async () => {
    const res = await getUserInfo()
    userInfo.value = res.data || {}
    localStorage.setItem(USER_KEY, JSON.stringify(userInfo.value))
    isAuthenticated.value = true
  }

  // 是否拥有某权限（超管拥有全部）
  const hasPermission = (perm) => {
    const info = userInfo.value
    if (!info) return false
    if (info.isSuperAdmin) return true
    const perms = info.permissions || []
    return perms.includes(perm) || perms.includes('*')
  }

  // 是否拥有任意一个权限
  const hasAnyPermission = (list) => list.some((p) => hasPermission(p))

  // 登出：通知后端清除 Cookie，并清理本地用户信息
  const logout = async () => {
    try {
      await apiLogout()
    } catch (e) {
      /* 忽略：即使后端失败也继续清理本地态 */
    }
    userInfo.value = null
    isAuthenticated.value = false
    localStorage.removeItem(USER_KEY)
  }

  // 初始化：通过后端探测登录态（Cookie 由浏览器自动携带）
  const init = async () => {
    if (initialized.value) return
    try {
      const res = await getUserInfo(true)
      userInfo.value = res.data || {}
      localStorage.setItem(USER_KEY, JSON.stringify(userInfo.value))
      isAuthenticated.value = true
    } catch (e) {
      isAuthenticated.value = false
    } finally {
      initialized.value = true
    }
  }

  const isSuperAdmin = computed(() => !!userInfo.value?.isSuperAdmin)

  return {
    userInfo,
    isAuthenticated,
    initialized,
    isSuperAdmin,
    login,
    handleCallback,
    refreshUserInfo,
    hasPermission,
    hasAnyPermission,
    logout,
    init
  }
})
