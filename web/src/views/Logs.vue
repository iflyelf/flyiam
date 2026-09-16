<template>
  <div class="logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>同步日志</span>
          <el-button :icon="Refresh" @click="loadLogs">刷新</el-button>
        </div>
      </template>

      <div class="filter-bar">
        <el-select v-model="filters.syncType" placeholder="同步类型" clearable style="width: 150px" @change="handleSearch">
          <el-option label="数据源同步" value="datasource" />
          <el-option label="Casdoor 同步" value="casdoor_user" />
          <el-option label="完整同步" value="full" />
        </el-select>
        <el-select v-model="filters.status" placeholder="状态" clearable style="width: 130px" @change="handleSearch">
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="运行中" value="running" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="Refresh" @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="logs" stripe style="width: 100%; margin-top: 16px">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="syncType" label="类型" width="130">
          <template #default="{ row }">{{ typeName(row.syncType) }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="totalCount" label="总数" width="90" />
        <el-table-column prop="successCount" label="成功" width="90">
          <template #default="{ row }"><span style="color: var(--ok-color)">{{ row.successCount }}</span></template>
        </el-table-column>
        <el-table-column prop="failedCount" label="失败" width="90">
          <template #default="{ row }">
            <span :style="{ color: row.failedCount > 0 ? 'var(--err-color)' : 'var(--text-secondary)' }">
              {{ row.failedCount }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="durationMs" label="耗时" width="100">
          <template #default="{ row }">{{ formatDuration(row.durationMs) }}</template>
        </el-table-column>
        <el-table-column prop="startedAt" label="开始时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.startedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleView(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top: 16px; justify-content: flex-end"
        @size-change="loadLogs"
        @current-change="loadLogs"
      />
    </el-card>

    <el-dialog v-model="detailVisible" title="同步日志详情" width="720px">
      <el-descriptions :column="2" border v-if="currentLog">
        <el-descriptions-item label="ID">{{ currentLog.id }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ typeName(currentLog.syncType) }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusType(currentLog.status)" size="small">{{ statusText(currentLog.status) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="触发方式">{{ currentLog.triggeredBy || '-' }}</el-descriptions-item>
        <el-descriptions-item label="总数">{{ currentLog.totalCount }}</el-descriptions-item>
        <el-descriptions-item label="成功数">{{ currentLog.successCount }}</el-descriptions-item>
        <el-descriptions-item label="失败数">{{ currentLog.failedCount }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ formatDuration(currentLog.durationMs) }}</el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatDateTime(currentLog.startedAt) }}</el-descriptions-item>
        <el-descriptions-item label="完成时间">{{ formatDateTime(currentLog.completedAt) }}</el-descriptions-item>
        <el-descriptions-item label="错误信息" :span="2" v-if="currentLog.errorMessage">
          <el-alert :title="currentLog.errorMessage" type="error" :closable="false" />
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Refresh, Search } from '@element-plus/icons-vue'
import { getSyncLogs } from '@/api/sync'
import { formatDateTime, formatDuration } from '@/utils/time'

const loading = ref(false)
const logs = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filters = ref({ syncType: '', status: '' })

const detailVisible = ref(false)
const currentLog = ref(null)

const loadLogs = async () => {
  loading.value = true
  try {
    const res = await getSyncLogs({
      syncType: filters.value.syncType,
      status: filters.value.status,
      page: page.value,
      pageSize: pageSize.value
    })
    const data = res.data || {}
    logs.value = data.list || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  page.value = 1
  loadLogs()
}
const handleReset = () => {
  filters.value = { syncType: '', status: '' }
  handleSearch()
}
const handleView = (row) => {
  currentLog.value = row
  detailVisible.value = true
}

const typeName = (t) => ({ datasource: '数据源同步', casdoor_user: 'Casdoor 同步', full: '完整同步' }[t] || t)
const statusType = (s) => ({ success: 'success', failed: 'danger', running: 'warning' }[s] || 'info')
const statusText = (s) => ({ success: '成功', failed: '失败', running: '运行中' }[s] || s)


onMounted(loadLogs)
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.filter-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}
</style>
