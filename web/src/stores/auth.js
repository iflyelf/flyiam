import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getUserInfo } from '@/api/auth'

const TOKEN_KEY = 'flyiam_token'
const USER_KEY = 'flyiam_user'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const userInfo = ref(JSON.parse(localStorage.getItem(USER_KEY) || 'null'))
  const isAuthenticated = ref(!!token.value)

  // 跳转到 Casdoor 登录（OAuth2 授权码模式，地址由后端动态生成）
  const login = () => {
    window.location.href = '/api/auth/login'
  }

  // 处理 Casdoor 回调，保存 token 并拉取用户权限
  const handleCallback = async (newToken) => {
    token.value = newToken
    localStorage.setItem(TOKEN_KEY, newToken)
    isAuthenticated.value = true
    try {
      const res = await getUserInfo()
      userInfo.value = res.data || {}
      localStorage.setItem(USER_KEY, JSON.stringify(userInfo.value))
    } catch (e) {
      userInfo.value = { name: '用户', permissions: [] }
    }
    return userInfo.value
  }

  // 刷新当前用户信息（含权限）
  const refreshUserInfo = async () => {
    if (!token.value) return
    const res = await getUserInfo()
    userInfo.value = res.data || {}
    localStorage.setItem(USER_KEY, JSON.stringify(userInfo.value))
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

  // 登出
  const logout = () => {
    token.value = ''
    userInfo.value = null
    isAuthenticated.value = false
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  // 初始化
  const init = () => {
    const savedToken = localStorage.getItem(TOKEN_KEY)
    const savedUser = localStorage.getItem(USER_KEY)
    if (savedToken) {
      token.value = savedToken
      userInfo.value = savedUser ? JSON.parse(savedUser) : null
      isAuthenticated.value = true
      // 后台刷新权限
      refreshUserInfo().catch(() => {})
    }
  }

  const isSuperAdmin = computed(() => !!userInfo.value?.isSuperAdmin)

  return {
    token,
    userInfo,
    isAuthenticated,
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
