<template>
  <div class="layout">
    <!-- 顶栏 -->
    <header class="topbar">
      <div class="brand">
        <img class="brand-logo" src="/favicon.svg" alt="logo" />
        <span class="brand-text">FlyIAM</span>
      </div>

      <!-- 桌面端导航 -->
      <nav class="tabs desktop-tabs">
        <router-link
          v-for="m in visibleMainMenus"
          :key="m.path"
          :to="m.path"
          class="tab"
          :class="{ active: route.path.startsWith(m.path) }"
        >
          <el-icon><component :is="m.icon" /></el-icon>
          <span>{{ m.label }}</span>
        </router-link>

        <!-- 人员组织（下拉子菜单） -->
        <el-dropdown v-if="visibleOrgMenus.length" class="org-dropdown" @command="handleOrgNav">
          <span class="tab" :class="{ active: isOrgActive }">
            <el-icon><OfficeBuilding /></el-icon>
            <span>人员组织</span>
            <el-icon class="org-arrow"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="o in visibleOrgMenus" :key="o.path" :command="o.path">
                {{ o.label }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </nav>

      <div class="topbar-actions">
        <!-- 主题切换 -->
        <div class="theme-switcher">
          <button
            v-for="t in themeStore.themes"
            :key="t.value"
            class="theme-btn"
            :class="{ active: themeStore.currentTheme === t.value }"
            :title="t.label"
            @click="themeStore.setTheme(t.value)"
          >
            {{ t.emoji }}
          </button>
        </div>

        <!-- 用户 -->
        <el-dropdown @command="handleCommand">
          <span class="user-info">
            <el-avatar :size="30" :src="avatar">
              {{ (displayName || '?').slice(0, 1) }}
            </el-avatar>
            <span class="username">{{ displayName }}</span>
            <el-icon class="user-arrow"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item disabled>
                <div class="user-info-dropdown">
                  <div class="user-name">{{ displayName }}</div>
                  <div class="user-email">{{ email }}</div>
                  <el-tag v-if="authStore.isSuperAdmin" type="danger" size="small" style="margin-top: 4px">
                    超级管理员
                  </el-tag>
                </div>
              </el-dropdown-item>
              <el-dropdown-item command="password" :icon="Lock">修改密码</el-dropdown-item>
              <el-dropdown-item divided command="logout" :icon="SwitchButton">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>

    <!-- 移动端导航（横向滚动） -->
    <nav class="tabs mobile-tabs">
      <router-link v-for="m in flatMenus" :key="m.path" :to="m.path" class="tab" :class="{ active: route.path.startsWith(m.path) }">
        <el-icon><component :is="m.icon" /></el-icon>
        <span>{{ m.label }}</span>
      </router-link>
    </nav>

    <!-- 主内容 -->
    <main class="layout-main">
      <div class="page-container">
        <router-view />
      </div>
    </main>

    <!-- 修改密码 -->
    <el-dialog v-model="pwdVisible" title="修改密码" width="440px">
      <el-form :model="pwdForm" label-width="90px">
        <el-form-item label="原密码" required>
          <el-input v-model="pwdForm.oldPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码" required>
          <el-input v-model="pwdForm.newPassword" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
        <el-form-item label="确认密码" required>
          <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdSaving" @click="handleChangePassword">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  UserFilled, SwitchButton, Odometer, User, Refresh, Document,
  Setting, ArrowDown, Timer, Connection, Lock, Grid, OfficeBuilding
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { changePassword } from '@/api/user'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const displayName = computed(() => authStore.userInfo?.name || authStore.userInfo?.username || '-')
const email = computed(() => authStore.userInfo?.email || '-')
const avatar = computed(() => {
  const url = authStore.userInfo?.avatar
  if (url) return url
  const ch = (displayName.value || '?').trim().slice(0, 1).toUpperCase()
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" rx="32" fill="#c8865a"/><text x="50%" y="54%" font-size="28" fill="#fff" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif">${ch}</text></svg>`
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
})

// 主导航（含权限要求）
const mainMenus = [
  { path: '/dashboard', label: '仪表盘', icon: Odometer },
  { path: '/sync', label: '数据同步', icon: Refresh, perm: 'sync:read' },
  { path: '/schedule', label: '定时任务', icon: Timer, perm: 'schedule:read' },
  { path: '/logs', label: '同步日志', icon: Document, perm: 'sync:read' },
  { path: '/casdoor', label: 'Casdoor 管理', icon: Connection, perm: 'casdoor:read' },
  { path: '/settings', label: '系统设置', icon: Setting }
]

// 人员组织子菜单
const orgMenus = [
  { path: '/users', label: '用户管理', icon: User, perm: 'user:read' },
  { path: '/teams', label: '团队管理', icon: Grid, perm: 'team:read' },
  { path: '/roles', label: '角色管理', icon: UserFilled, perm: 'role:read' }
]

const visibleMainMenus = computed(() => mainMenus.filter((m) => !m.perm || authStore.hasPermission(m.perm)))
const visibleOrgMenus = computed(() => orgMenus.filter((m) => !m.perm || authStore.hasPermission(m.perm)))
const flatMenus = computed(() => [...visibleMainMenus.value, ...visibleOrgMenus.value])

const isOrgActive = computed(() => orgMenus.some((o) => route.path.startsWith(o.path)))
const handleOrgNav = (path) => router.push(path)

// ---------------- 修改密码 ----------------
const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })

const handleChangePassword = async () => {
  if (!pwdForm.value.oldPassword || !pwdForm.value.newPassword) {
    ElMessage.warning('请填写完整')
    return
  }
  if (pwdForm.value.newPassword !== pwdForm.value.confirmPassword) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  pwdSaving.value = true
  try {
    await changePassword(pwdForm.value.oldPassword, pwdForm.value.newPassword)
    ElMessage.success('密码修改成功')
    pwdVisible.value = false
    pwdForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
  } finally {
    pwdSaving.value = false
  }
}

const handleCommand = async (command) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      authStore.logout()
      ElMessage.success('已退出登录')
      router.push('/login')
    } catch (error) {
      /* 取消 */
    }
  } else if (command === 'password') {
    pwdVisible.value = true
  }
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 20px;
  background: color-mix(in srgb, var(--card-bg) 88%, transparent);
  backdrop-filter: saturate(180%) blur(14px);
  -webkit-backdrop-filter: saturate(180%) blur(14px);
  border-bottom: 1px solid var(--border-color);
  box-shadow: var(--shadow);
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.brand-logo {
  width: 30px;
  height: 30px;
  border-radius: 9px;
}

.brand-text {
  font-size: 17px;
  font-weight: 700;
  color: var(--primary-color);
  white-space: nowrap;
}

.tabs {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
}

.tabs::-webkit-scrollbar {
  display: none;
}

.desktop-tabs {
  flex: 1;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  text-decoration: none;
  white-space: nowrap;
  transition: all 0.2s;
}

.tab:hover {
  background: color-mix(in srgb, var(--primary-color) 10%, transparent);
}

.tab.active {
  background: var(--primary-color);
  color: #fffaf3;
}

.mobile-tabs {
  display: none;
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.theme-switcher {
  display: flex;
  gap: 4px;
  background: var(--panel-bg);
  padding: 4px;
  border-radius: var(--radius);
  border: 1px solid var(--border-color);
}

.theme-btn {
  background: transparent;
  border: none;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  font-size: 15px;
  line-height: 1;
  cursor: pointer;
  opacity: 0.55;
  transition: all 0.2s;
}

.theme-btn:hover {
  opacity: 1;
  background: color-mix(in srgb, var(--primary-color) 12%, transparent);
}

.theme-btn.active {
  opacity: 1;
  background: color-mix(in srgb, var(--primary-color) 22%, transparent);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: var(--text-color);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  transition: background 0.2s;
}

.user-info:hover {
  background: color-mix(in srgb, var(--primary-color) 10%, transparent);
}

.username {
  font-size: 14px;
}

.user-arrow {
  font-size: 12px;
  color: var(--text-secondary);
}

.user-info-dropdown {
  padding: 4px 0;
}

.user-name {
  font-weight: 600;
  color: var(--text-color);
}

.user-email {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.layout-main {
  flex: 1;
  overflow-x: hidden;
}

.layout-main .page-container {
  max-width: 1400px;
  margin: 0 auto;
}

@media (max-width: 860px) {
  .desktop-tabs {
    display: none;
  }

  .topbar {
    padding: 10px 14px;
  }

  .mobile-tabs {
    display: flex;
    position: sticky;
    top: 51px;
    z-index: 190;
    padding: 8px 12px;
    gap: 6px;
    background: color-mix(in srgb, var(--card-bg) 92%, transparent);
    backdrop-filter: saturate(180%) blur(14px);
    -webkit-backdrop-filter: saturate(180%) blur(14px);
    border-bottom: 1px solid var(--border-color);
  }

  .brand-text {
    font-size: 15px;
  }

  .username {
    display: none;
  }
}

@media (max-width: 480px) {
  .theme-switcher {
    gap: 2px;
    padding: 3px;
  }

  .theme-btn {
    padding: 3px 6px;
    font-size: 13px;
  }
}
</style>
