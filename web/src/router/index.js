import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/components/Layout.vue'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', requiresAuth: false }
  },
  {
    path: '/callback',
    name: 'Callback',
    component: () => import('@/views/Callback.vue'),
    meta: { title: '登录中', requiresAuth: false }
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: '/dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'DataLine', requiresAuth: true }
      },
      {
        path: '/users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { title: '用户管理', icon: 'User', requiresAuth: true }
      },
      {
        path: '/user-fields',
        name: 'UserFields',
        component: () => import('@/views/UserFields.vue'),
        meta: { title: '用户字段', icon: 'SetUp', requiresAuth: true }
      },
      {
        path: '/teams',
        name: 'Teams',
        component: () => import('@/views/Teams.vue'),
        meta: { title: '团队管理', icon: 'Grid', requiresAuth: true }
      },
      {
        path: '/roles',
        name: 'Roles',
        component: () => import('@/views/Roles.vue'),
        meta: { title: '角色管理', icon: 'Avatar', requiresAuth: true }
      },
      {
        path: '/sync',
        name: 'Sync',
        component: () => import('@/views/Sync.vue'),
        meta: { title: '数据同步', icon: 'Refresh', requiresAuth: true }
      },
      {
        path: '/schedule',
        name: 'Schedule',
        component: () => import('@/views/Schedule.vue'),
        meta: { title: '定时任务', icon: 'Timer', requiresAuth: true }
      },
      {
        path: '/logs',
        name: 'Logs',
        component: () => import('@/views/Logs.vue'),
        meta: { title: '同步日志', icon: 'Document', requiresAuth: true }
      },
      {
        path: '/casdoor',
        name: 'CasdoorAdmin',
        component: () => import('@/views/Casdoor.vue'),
        meta: { title: 'Casdoor 管理', icon: 'Connection', requiresAuth: true }
      },
      {
        path: '/settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: '系统设置', icon: 'Setting', requiresAuth: true }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  
  // 初始化 auth store
  if (!authStore.isAuthenticated) {
    authStore.init()
  }

  // 需要认证的路由
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } 
  // 已登录访问登录页，重定向到首页
  else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/')
  } 
  else {
    next()
  }
})

export default router
