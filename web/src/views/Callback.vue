<template>
  <div class="callback-container">
    <el-card class="callback-card">
      <div class="callback-content">
        <el-icon v-if="status === 'loading'" class="is-loading" :size="40"><Loading /></el-icon>
        <el-icon v-else-if="status === 'success'" :size="40" color="var(--ok-color)"><CircleCheck /></el-icon>
        <el-icon v-else :size="40" color="var(--err-color)"><CircleClose /></el-icon>
        <p class="callback-text">{{ message }}</p>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Loading, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const status = ref('loading')
const message = ref('正在完成登录...')

onMounted(async () => {
  const token = route.query.token
  if (!token) {
    status.value = 'error'
    message.value = '登录失败：缺少凭证'
    setTimeout(() => router.push('/login'), 2000)
    return
  }
  try {
    await authStore.handleCallback(token)
    status.value = 'success'
    message.value = '登录成功，正在跳转...'
    setTimeout(() => router.push('/'), 800)
  } catch (e) {
    status.value = 'error'
    message.value = '登录失败：' + e.message
    setTimeout(() => router.push('/login'), 2000)
  }
})
</script>

<style scoped>
.callback-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: var(--bg-color);
  background-image: var(--bg-grad);
}
.callback-card {
  width: 360px;
}
.callback-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 20px 0;
}
.callback-text {
  color: var(--text-color);
  font-size: 15px;
}
</style>
