<template>
  <div class="login-container">
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

    <el-card class="login-card">
      <template #header>
        <div class="card-header">
          <img class="login-logo" src="/favicon.svg" alt="logo" />
          <h2>{{ appName }}</h2>
          <p>统一用户管理系统</p>
        </div>
      </template>

      <div class="login-content">
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          class="login-button"
          @click="handleLogin"
        >
          <el-icon class="el-icon--left"><User /></el-icon>
          使用 Casdoor 登录
        </el-button>

        <div class="login-tips">
          <el-alert
            title="使用 Casdoor 统一认证平台登录，支持 SSO 单点登录"
            type="info"
            :closable="false"
            show-icon
          />
        </div>

        <div class="login-info">
          <el-divider />
          <p class="info-text">
            <el-icon><Lock /></el-icon>
            账号由管理员开通，首次登录请联系管理员
          </p>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { User, Lock } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { getAuthConfig } from '@/api/auth'

const authStore = useAuthStore()
const themeStore = useThemeStore()

const appName = ref('FlyIAM')
const loading = ref(false)

const handleLogin = () => {
  loading.value = true
  authStore.login()
}

onMounted(async () => {
  try {
    const res = await getAuthConfig()
    if (res.data?.appName) appName.value = res.data.appName
  } catch (e) {
    // 忽略
  }
})
</script>

<style scoped>
.login-container {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 20px;
  background-color: var(--bg-color);
  background-image: var(--bg-grad);
}
.theme-switcher {
  position: absolute;
  top: 20px;
  right: 20px;
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
.login-card {
  width: 450px;
  max-width: 100%;
}
.card-header {
  text-align: center;
  padding: 10px 0;
}
.login-logo {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  margin-bottom: 10px;
}
.card-header h2 {
  margin: 0 0 8px 0;
  color: var(--primary-color);
  font-size: 26px;
  font-weight: 700;
}
.card-header p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}
.login-content {
  padding: 10px 0;
}
.login-button {
  width: 100%;
  height: 46px;
  font-size: 16px;
}
.login-tips {
  margin-top: 16px;
}
.login-info {
  margin-top: 12px;
}
.info-text {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 13px;
  margin: 6px 0;
}
@media (max-width: 768px) {
  .login-card {
    width: 100%;
  }
  .card-header h2 {
    font-size: 22px;
  }
}
</style>
