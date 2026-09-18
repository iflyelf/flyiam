<template>
  <div class="settings">
    <el-row :gutter="16">
      <el-col :xs="24" :lg="12">
        <el-card>
          <template #header><span>系统信息</span></template>
          <el-descriptions :column="1" border v-loading="loading">
            <el-descriptions-item label="系统名称">{{ sys.appName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Casdoor 用户总数">{{ stats.total || 0 }}</el-descriptions-item>
            <el-descriptions-item label="受保护用户">{{ (stats.protected || []).join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item label="数据源数量">{{ dataSourceCount }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="12">
        <el-card>
          <template #header><span>数据源</span></template>
          <el-table :data="dataSources" size="small" style="width: 100%">
            <el-table-column prop="name" label="名称" width="130" />
            <el-table-column prop="url" label="地址" min-width="200" show-overflow-tooltip />
            <el-table-column prop="enabled" label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
                  {{ row.enabled ? '启用' : '停用' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 12px">
            <el-button type="primary" @click="$router.push('/sync')">管理数据源</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 可页面配置项（DB 优先 / env 兜底，保存即时生效） -->
    <el-card style="margin-top: 16px" v-if="canWrite">
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>系统配置</span>
            <el-tag type="info" size="small">保存后即时生效（Casdoor 连接变更会自动重建客户端）</el-tag>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="loadSettings">刷新</el-button>
            <el-button type="primary" :loading="savingSettings" @click="saveSettings">保存配置</el-button>
          </div>
        </div>
      </template>

      <el-collapse v-model="activeGroups">
        <el-collapse-item v-for="(items, group) in groupedSettings" :key="group" :name="group">
          <template #title>
            <span class="group-title">{{ group }}</span>
          </template>
          <el-form label-width="200px" label-position="right">
            <el-form-item v-for="it in items" :key="it.key" :label="it.label">
              <el-switch v-if="it.type === 'bool'" v-model="settingsForm[it.key]" />
              <el-input-number v-else-if="it.type === 'int'" v-model="settingsForm[it.key]" :controls="false" style="width: 220px" />
              <el-input
                v-else
                v-model="settingsForm[it.key]"
                :type="it.secret ? 'password' : 'text'"
                :show-password="it.secret"
                :placeholder="it.secret ? '留空/保持占位符表示不修改' : ''"
                style="max-width: 420px"
              />
              <span class="key-tip">{{ it.key }}</span>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </el-card>

    <el-card style="margin-top: 16px">
      <template #header><span>关于</span></template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="项目名称">FlyIAM</el-descriptions-item>
        <el-descriptions-item label="项目描述">统一用户管理系统</el-descriptions-item>
        <el-descriptions-item label="许可证">MIT License</el-descriptions-item>
        <el-descriptions-item label="前端框架">Vue 3 + Element Plus</el-descriptions-item>
        <el-descriptions-item label="后端框架">Go + go-zero</el-descriptions-item>
        <el-descriptions-item label="项目地址">
          <el-link href="https://github.com/iflyelf/flyiam" target="_blank">github.com/iflyelf/flyiam</el-link>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getAuthConfig } from '@/api/auth'
import { getUserStats } from '@/api/user'
import { listDataSources } from '@/api/datasource'
import { listSettings, updateSettings } from '@/api/setting'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canWrite = computed(() => authStore.hasPermission('setting:write'))

const loading = ref(false)
const sys = ref({})
const stats = ref({})
const dataSources = ref([])
const dataSourceCount = ref(0)

// 系统配置
const savingSettings = ref(false)
const settingsForm = ref({})
const settingsItems = ref([])
const activeGroups = ref([])

const groupedSettings = computed(() => {
  const out = {}
  for (const it of settingsItems.value) {
    if (!out[it.group]) out[it.group] = []
    out[it.group].push(it)
  }
  // 默认展开第一个分组
  if (activeGroups.value.length === 0 && Object.keys(out).length > 0) {
    activeGroups.value = [Object.keys(out)[0]]
  }
  return out
})

const loadSettings = async () => {
  try {
    const res = await listSettings()
    settingsItems.value = res || []
    const form = {}
    for (const it of settingsItems.value) {
      form[it.key] = it.type === 'bool' ? it.value === 'true' : it.value
    }
    settingsForm.value = form
  } catch (e) {
    /* 拦截器已提示 */
  }
}

const saveSettings = async () => {
  savingSettings.value = true
  try {
    // bool 转字符串；secret 占位符 ****** 原样提交（后端识别为保持原值）
    const payload = {}
    for (const it of settingsItems.value) {
      const v = settingsForm.value[it.key]
      payload[it.key] = it.type === 'bool' ? String(!!v) : String(v ?? '')
    }
    await updateSettings(payload)
    ElMessage.success('配置已保存并生效')
    loadSettings()
  } finally {
    savingSettings.value = false
  }
}

const load = async () => {
  loading.value = true
  try {
    const [cfgRes, statsRes, dsRes] = await Promise.all([
      getAuthConfig().catch(() => ({ data: {} })),
      getUserStats().catch(() => ({ data: {} })),
      listDataSources().catch(() => ({ data: [] }))
    ])
    sys.value = cfgRes.data || {}
    stats.value = statsRes.data || {}
    dataSources.value = dsRes.data || []
    dataSourceCount.value = dataSources.value.length
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  loadSettings()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.group-title {
  font-weight: 600;
}
.key-tip {
  margin-left: 12px;
  color: var(--text-secondary);
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
</style>
