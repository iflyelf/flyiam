<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <el-row :gutter="16">
      <el-col :xs="12" :sm="12" :lg="6" v-for="stat in stats" :key="stat.title">
        <el-card class="stat-card" shadow="hover" @click="stat.action && stat.action()">
          <div class="stat-content">
            <div class="stat-icon" :style="{ background: stat.color }">
              <el-icon :size="26"><component :is="stat.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stat.value }}</div>
              <div class="stat-title">{{ stat.title }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <!-- 最近同步记录 -->
      <el-col :xs="24" :lg="16">
        <el-card class="panel">
          <template #header>
            <div class="card-header">
              <span>最近同步记录</span>
              <el-button text type="primary" @click="$router.push('/logs')">查看更多</el-button>
            </div>
          </template>
          <el-table v-loading="loadingLogs" :data="recentLogs" style="width: 100%" :show-header="true">
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatDateTime(row.startedAt) }}</template>
            </el-table-column>
            <el-table-column label="类型" width="120">
              <template #default="{ row }">{{ typeName(row.syncType) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small" effect="plain">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="成功/总数">
              <template #default="{ row }">{{ row.successCount }}/{{ row.totalCount }}</template>
            </el-table-column>
            <el-table-column label="耗时" width="90">
              <template #default="{ row }">{{ formatDuration(row.durationMs) }}</template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loadingLogs && recentLogs.length === 0" description="暂无同步记录" :image-size="80" />
        </el-card>
      </el-col>

      <!-- 右侧：同步状态 + 快速操作 -->
      <el-col :xs="24" :lg="8">
        <el-card class="panel">
          <template #header>
            <div class="card-header">
              <span>同步状态</span>
              <el-button text type="primary" :icon="Refresh" @click="loadProgress">刷新</el-button>
            </div>
          </template>
          <div class="progress-list">
            <div class="progress-item" v-for="p in progressList" :key="p.type">
              <div class="progress-item-head">
                <span class="progress-label">{{ p.label }}</span>
                <el-tag :type="statusType(p.status)" size="small" effect="plain">
                  {{ statusText(p.status) }}
                </el-tag>
              </div>
              <div class="progress-item-meta">
                <span>{{ p.successCount }}/{{ p.totalCount }} 成功</span>
                <span class="muted">{{ p.startedAt ? formatDateTime(p.startedAt) : '尚未执行' }}</span>
              </div>
            </div>
          </div>
        </el-card>

        <el-card class="panel" style="margin-top: 16px">
          <template #header><span>快速操作</span></template>
          <div class="quick-actions">
            <el-button type="primary" :icon="Download" :loading="syncing.datasource" :disabled="anyRunning" @click="doSync('datasource', syncDataSource)">
              数据源同步
            </el-button>
            <el-button type="warning" :icon="Refresh" :loading="syncing.full" :disabled="anyRunning" @click="doSync('full', () => syncFull())">
              完整同步
            </el-button>
            <el-button :icon="User" @click="$router.push('/users')">查看用户</el-button>
            <el-button :icon="Timer" @click="$router.push('/schedule')">定时任务</el-button>
          </div>
          <el-progress
            v-if="currentRunning"
            style="margin-top: 12px"
            :percentage="progressPercent"
            :stroke-width="8"
            :format="() => `${currentRunning.successCount}/${currentRunning.totalCount || 0}`"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { User, UserFilled, Warning, Refresh, Download, Timer, Connection } from '@element-plus/icons-vue'
import { getUserStats } from '@/api/user'
import { listDataSources } from '@/api/datasource'
import { listTeams } from '@/api/rbac'
import { getSyncLogs, getSyncProgress, syncDataSource, syncFull } from '@/api/sync'
import { formatDateTime, formatDuration } from '@/utils/time'
import { useRouter } from 'vue-router'

const router = useRouter()
const statData = ref({ total: 0, protected: [] })
const dataSourceCount = ref(0)
const teamCount = ref(0)
const recentLogs = ref([])
const loadingLogs = ref(false)
const syncing = ref({ datasource: false, full: false })
const progressList = ref([])

// 同步运行状态
const anyRunning = computed(() => progressList.value.some((p) => p.status === 'running'))
const currentRunning = computed(() => progressList.value.find((p) => p.status === 'running') || null)
const progressPercent = computed(() => {
  const c = currentRunning.value
  if (!c || !c.totalCount) return 0
  return Math.min(100, Math.floor((c.successCount / c.totalCount) * 100))
})

const stats = computed(() => [
  {
    title: 'Casdoor 用户总数',
    value: statData.value.total || 0,
    icon: UserFilled,
    color: 'linear-gradient(135deg,#667eea,#764ba2)',
    action: () => router.push('/users')
  },
  {
    title: '受保护用户',
    value: (statData.value.protected || []).length,
    icon: Warning,
    color: 'linear-gradient(135deg,#f093fb,#f5576c)',
    action: () => router.push('/casdoor')
  },
  {
    title: '数据源数量',
    value: dataSourceCount.value,
    icon: Connection,
    color: 'linear-gradient(135deg,#43e97b,#38f9d7)',
    action: () => router.push('/sync')
  },
  {
    title: '团队数量',
    value: teamCount.value,
    icon: User,
    color: 'linear-gradient(135deg,#fa709a,#fee140)',
    action: () => router.push('/teams')
  }
])

const loadStats = async () => {
  const [statsRes, dsRes, teamRes] = await Promise.all([
    getUserStats().catch(() => ({ data: {} })),
    listDataSources().catch(() => ({ data: [] })),
    listTeams().catch(() => ({ data: {} }))
  ])
  statData.value = statsRes.data || { total: 0, protected: [] }
  dataSourceCount.value = (dsRes.data || []).length
  teamCount.value = teamRes.data?.total || 0
}

const loadRecentLogs = async () => {
  loadingLogs.value = true
  try {
    const res = await getSyncLogs({ page: 1, pageSize: 5 })
    recentLogs.value = res.data?.list || []
  } finally {
    loadingLogs.value = false
  }
}

const loadProgress = async () => {
  const res = await getSyncProgress().catch(() => ({ data: {} }))
  const data = res.data || {}
  const labelMap = { datasource: '数据源同步', full: '完整同步' }
  progressList.value = Object.keys(data).map((k) => ({
    type: k,
    label: labelMap[k] || k,
    status: data[k].status,
    successCount: data[k].successCount,
    totalCount: data[k].totalCount,
    startedAt: data[k].startedAt
  }))
  if (progressList.value.length === 0) {
    progressList.value = [
      { type: 'datasource', label: '数据源同步', status: '', successCount: 0, totalCount: 0, startedAt: null },
      { type: 'full', label: '完整同步', status: '', successCount: 0, totalCount: 0, startedAt: null }
    ]
  }
}

let pollTimer = null
const startPolling = () => {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    await Promise.all([loadProgress(), loadRecentLogs(), loadStats()])
    if (!anyRunning.value) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }, 5000)
}

const doSync = async (key, fn) => {
  syncing.value[key] = true
  try {
    await fn()
    ElMessage.success('同步任务已启动')
    await loadProgress()
    startPolling()
  } catch (e) {
    // 错误提示由请求拦截器处理
  } finally {
    syncing.value[key] = false
  }
}

const typeName = (t) => ({ datasource: '数据源同步', casdoor_user: 'Casdoor 同步', full: '完整同步' }[t] || t)
const statusType = (s) => ({ success: 'success', failed: 'danger', running: 'warning' }[s] || 'info')
const statusText = (s) => ({ success: '成功', failed: '失败', running: '运行中' }[s] || '未执行')

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

onMounted(async () => {
  loadStats()
  loadRecentLogs()
  await loadProgress()
  if (anyRunning.value) startPolling()
})
</script>

<style scoped>
.stat-card {
  cursor: pointer;
}
.stat-card :deep(.el-card__body) {
  padding: 16px;
}
.stat-content {
  display: flex;
  align-items: center;
  gap: 14px;
}
.stat-icon {
  width: 50px;
  height: 50px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}
.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-color);
  line-height: 1.2;
}
.stat-title {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 2px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.panel :deep(.el-card__body) {
  padding: 16px;
}
.progress-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.progress-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.progress-label {
  font-weight: 500;
  color: var(--text-color);
}
.progress-item-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}
.muted {
  color: var(--text-secondary);
}
.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.quick-actions .el-button {
  margin-left: 0;
}
</style>
