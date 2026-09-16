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
import { ref, onMounted } from 'vue'
import { getAuthConfig } from '@/api/auth'
import { getUserStats } from '@/api/user'
import { listDataSources } from '@/api/datasource'

const loading = ref(false)
const sys = ref({})
const stats = ref({})
const dataSources = ref([])
const dataSourceCount = ref(0)

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

onMounted(load)
</script>

<style scoped>
</style>
