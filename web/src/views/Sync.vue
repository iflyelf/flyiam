<template>
  <div class="sync">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>数据源配置</span>
          <el-button type="primary" :icon="Plus" @click="openCreate">新增数据源</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="dataSources" stripe style="width: 100%">
        <el-table-column prop="name" label="名称" width="140" />
        <el-table-column prop="type" label="类型" width="100" />
        <el-table-column prop="url" label="地址" min-width="240" show-overflow-tooltip />
        <el-table-column prop="method" label="方法" width="80" />
        <el-table-column prop="syncInterval" label="同步间隔" width="100" />
        <el-table-column prop="priority" label="优先级" width="90" />
        <el-table-column prop="enabled" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleTestRow(row)">测试</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span>数据同步</span>
          <el-button :icon="Refresh" @click="loadProgress">刷新进度</el-button>
        </div>
      </template>

      <div class="sync-actions">
        <el-button type="primary" :icon="Download" :loading="syncing.datasource" :disabled="anyRunning" @click="handleSyncDataSource">
          从数据源同步
        </el-button>
        <el-button type="warning" :icon="Refresh" :loading="syncing.full" :disabled="anyRunning" @click="handleSyncFull">
          完整同步
        </el-button>
      </div>

      <!-- 运行中进度 -->
      <div class="running-box" v-if="currentRunning">
        <div class="running-head">
          <span class="running-title">
            <el-icon class="is-loading"><Loading /></el-icon>
            正在同步（{{ statusText(currentRunning.status) }}）
          </span>
          <span class="running-count">{{ currentRunning.successCount }} / {{ currentRunning.totalCount || '…' }}</span>
        </div>
        <el-progress
          :percentage="progressPercent"
          :stroke-width="10"
          :status="currentRunning.status === 'failed' ? 'exception' : ''"
        />
      </div>

      <el-descriptions :column="1" border style="margin-top: 16px" v-if="progressList.length">
        <el-descriptions-item v-for="p in progressList" :key="p.type" :label="p.label">
          <el-tag :type="statusType(p.status)" size="small">{{ statusText(p.status) }}</el-tag>
          <span class="progress-meta">
            成功 {{ p.successCount }}/{{ p.totalCount }} · 失败 {{ p.failedCount }}
            <template v-if="p.startedAt"> · {{ formatDateTime(p.startedAt) }}</template>
          </span>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 数据源编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑数据源' : '新增数据源'" width="620px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如 personnel" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type">
            <el-option label="HTTP API" value="httpapi" />
          </el-select>
        </el-form-item>
        <el-form-item label="接口地址" required>
          <el-input v-model="form.url" placeholder="http://host:port/api/xxx" />
        </el-form-item>
        <el-form-item label="请求方法">
          <el-select v-model="form.method">
            <el-option label="POST" value="POST" />
            <el-option label="GET" value="GET" />
          </el-select>
        </el-form-item>
        <el-form-item label="认证方式">
          <el-select v-model="form.authType">
            <el-option label="无" value="none" />
            <el-option label="Bearer Token" value="bearer" />
            <el-option label="Basic Auth" value="basic" />
            <el-option label="API Key" value="apikey" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.authType === 'bearer' || form.authType === 'apikey'" label="Token">
          <el-input v-model="form.authToken" type="password" show-password />
        </el-form-item>
        <template v-if="form.authType === 'basic'">
          <el-form-item label="用户名">
            <el-input v-model="form.authUsername" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.authPassword" type="password" show-password />
          </el-form-item>
        </template>
        <el-form-item label="每页大小">
          <el-input-number v-model="form.pageSize" :min="1" :max="100000" />
        </el-form-item>
        <el-form-item label="超时(秒)">
          <el-input-number v-model="form.timeout" :min="1" :max="600" />
        </el-form-item>
        <el-form-item label="同步间隔">
          <el-input v-model="form.syncInterval" placeholder="如 6h" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="0" :max="1000" />
        </el-form-item>
        <el-form-item label="字段映射">
          <div class="mapping-editor">
            <div class="mapping-hint">
              外部字段名 → 内部字段键；留空则使用内置默认字段名（DOMACT/NAME/...）。数据源字段变化只需改此处，无需改代码。
            </div>
            <div v-for="(m, idx) in mappingRows" :key="idx" class="mapping-row">
              <el-input v-model="m.external" placeholder="外部字段名，如 DOMACT" style="width: 200px" />
              <span class="arrow">→</span>
              <el-select
                v-model="m.internal"
                filterable
                allow-create
                default-first-option
                placeholder="内部字段键"
                style="width: 220px"
              >
                <el-option v-for="k in internalKeyOptions" :key="k.value" :label="k.label" :value="k.value" />
              </el-select>
              <el-button link type="danger" :icon="Delete" @click="mappingRows.splice(idx, 1)" />
            </div>
            <el-button link type="primary" :icon="Plus" @click="mappingRows.push({ external: '', internal: '' })">
              添加映射
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button :loading="testing" @click="handleTestForm">测试连接</el-button>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 测试结果弹窗 -->
    <el-dialog v-model="testVisible" title="连接测试结果" width="700px">
      <el-alert :type="testResult.connected ? 'success' : 'error'" :closable="false" show-icon
        :title="testResult.connected ? `连接成功，共 ${testResult.total} 条数据` : testResult.message" />
      <el-table v-if="testResult.connected" :data="testResult.sample" size="small" style="margin-top: 12px">
        <el-table-column prop="DomainAccount" label="域账号" width="120" />
        <el-table-column prop="Name" label="姓名" width="100" />
        <el-table-column prop="DeptNameLv1" label="一级部门" min-width="140" show-overflow-tooltip />
        <el-table-column prop="WorkEmail" label="邮箱" min-width="160" show-overflow-tooltip />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Download, Loading, Delete } from '@element-plus/icons-vue'
import { listUserFields } from '@/api/userField'
import {
  listDataSources,
  createDataSource,
  updateDataSource,
  deleteDataSource,
  testDataSource,
  testDataSourceConfig
} from '@/api/datasource'
import { syncDataSource, syncFull, getSyncProgress } from '@/api/sync'
import { formatDateTime } from '@/utils/time'

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const dataSources = ref([])

const dialogVisible = ref(false)
const testVisible = ref(false)
const testResult = ref({})

const syncing = ref({ datasource: false, casdoor: false, full: false })
const progressList = ref([])

const emptyForm = () => ({
  id: null,
  name: '',
  type: 'httpapi',
  url: '',
  method: 'POST',
  authType: 'none',
  authToken: '',
  authUsername: '',
  authPassword: '',
  timeout: 30,
  syncInterval: '6h',
  autoSync: true,
  priority: 90,
  pageSize: 1000,
  enabled: true,
  remark: ''
})
const form = ref(emptyForm())

// 字段映射编辑（外部字段名 → 内部字段键）
const mappingRows = ref([])
const userFields = ref([])

// 内置字段键（与后端 convertWithMapping 的标准字段一致）
const builtinKeys = [
  { value: 'domainAccount', label: '域账号（唯一标识，必填）' },
  { value: 'name', label: '姓名' },
  { value: 'employeeCode', label: '工号' },
  { value: 'workEmail', label: '邮箱' },
  { value: 'phone', label: '手机号' },
  { value: 'compileType', label: '编制类型' },
  { value: 'superiorAccount', label: '上级账号' },
  { value: 'deptNameLv0', label: '零级部门' },
  { value: 'deptNameLv1', label: '一级部门' },
  { value: 'deptNameLv2', label: '二级部门' },
  { value: 'deptIdLv0', label: '零级部门ID' },
  { value: 'deptIdLv1', label: '一级部门ID' },
  { value: 'deptIdLv2', label: '二级部门ID' }
]

const internalKeyOptions = computed(() => {
  const opts = [...builtinKeys]
  for (const f of userFields.value) {
    if (!opts.some((o) => o.value === f.fieldKey)) {
      opts.push({ value: f.fieldKey, label: `${f.label}（自定义：${f.fieldKey}）` })
    }
  }
  return opts
})

const loadUserFields = async () => {
  try {
    const res = await listUserFields()
    userFields.value = res.data || []
  } catch (e) {
    userFields.value = []
  }
}

// parseMapping 解析 fieldMapping JSON 字符串为编辑行
const parseMapping = (raw) => {
  if (!raw) return []
  try {
    const obj = JSON.parse(raw)
    return Object.keys(obj).map((k) => ({ external: k, internal: obj[k] }))
  } catch (e) {
    return []
  }
}

// buildMapping 将编辑行组装为 JSON 字符串
const buildMapping = () => {
  const obj = {}
  for (const r of mappingRows.value) {
    const ext = (r.external || '').trim()
    const internal = (r.internal || '').trim()
    if (ext && internal) obj[ext] = internal
  }
  return Object.keys(obj).length ? JSON.stringify(obj) : ''
}

const loadDataSources = async () => {
  loading.value = true
  try {
    const res = await listDataSources()
    dataSources.value = res.data || []
  } finally {
    loading.value = false
  }
}

const loadProgress = async () => {
  const res = await getSyncProgress()
  const data = res.data || {}
  const labelMap = { datasource: '数据源同步', casdoor_user: 'Casdoor 同步', full: '完整同步' }
  progressList.value = Object.keys(data).map((k) => ({
    type: k,
    label: labelMap[k] || k,
    status: data[k].status,
    successCount: data[k].successCount,
    totalCount: data[k].totalCount,
    failedCount: data[k].failedCount,
    startedAt: data[k].startedAt
  }))
}

const openCreate = () => {
  form.value = emptyForm()
  mappingRows.value = []
  dialogVisible.value = true
}

const openEdit = (row) => {
  form.value = { ...emptyForm(), ...row }
  mappingRows.value = parseMapping(row.fieldMapping)
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!form.value.name || !form.value.url) {
    ElMessage.warning('请填写名称与接口地址')
    return
  }
  form.value.fieldMapping = buildMapping()
  saving.value = true
  try {
    if (form.value.id) {
      await updateDataSource(form.value.id, form.value)
      ElMessage.success('已更新')
    } else {
      await createDataSource(form.value)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    loadDataSources()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  await ElMessageBox.confirm(`确定删除数据源「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteDataSource(row.id)
  ElMessage.success('已删除')
  loadDataSources()
}

const handleTestForm = async () => {
  testing.value = true
  try {
    const res = await testDataSourceConfig(form.value)
    testResult.value = res.data || {}
    testVisible.value = true
  } catch (e) {
    testResult.value = { connected: false, message: e.message }
    testVisible.value = true
  } finally {
    testing.value = false
  }
}

const handleTestRow = async (row) => {
  try {
    const res = await testDataSource(row.id)
    testResult.value = res.data || {}
    testVisible.value = true
  } catch (e) {
    testResult.value = { connected: false, message: e.message }
    testVisible.value = true
  }
}

let pollTimer = null

// 启动轮询，实时刷新进度，直到同步结束
const startPolling = () => {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    await loadProgress()
    if (!currentRunning.value && !anyRunning.value) {
      stopPolling()
    }
  }, 3000)
}
const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const runSync = async (key, fn) => {
  syncing.value[key] = true
  try {
    await fn()
    ElMessage.success('同步任务已启动')
    await loadProgress()
    startPolling()
  } catch (e) {
    // 错误提示由请求拦截器统一处理
  } finally {
    syncing.value[key] = false
  }
}

const handleSyncDataSource = () => runSync('datasource', syncDataSource)
const handleSyncFull = () => runSync('full', () => syncFull())

const statusType = (s) => ({ success: 'success', failed: 'danger', running: 'warning' }[s] || 'info')
const statusText = (s) => ({ success: '成功', failed: '失败', running: '运行中' }[s] || '未执行')

// 是否有任务运行中
const anyRunning = computed(() => progressList.value.some((p) => p.status === 'running'))
const currentRunning = computed(() => progressList.value.find((p) => p.status === 'running') || null)
const progressPercent = computed(() => {
  const c = currentRunning.value
  if (!c || !c.totalCount) return 0
  return Math.min(100, Math.floor((c.successCount / c.totalCount) * 100))
})


onMounted(async () => {
  loadDataSources()
  loadUserFields()
  await loadProgress()
  // 若已有任务在运行（如定时触发或其他入口触发），自动开始轮询
  if (anyRunning.value) startPolling()
})

onUnmounted(stopPolling)
</script>

<style scoped>
.mapping-editor {
  width: 100%;
}
.mapping-hint {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 8px;
}
.mapping-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.mapping-row .arrow {
  color: var(--text-secondary);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.sync-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.running-box {
  margin-top: 16px;
  padding: 14px 16px;
  border-radius: var(--radius);
  background: color-mix(in srgb, var(--primary-color) 6%, var(--card-bg));
  border: 1px solid var(--border-color);
}
.running-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.running-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: var(--text-color);
}
.running-count {
  font-size: 13px;
  color: var(--text-secondary);
}
.progress-meta {
  margin-left: 12px;
  color: var(--text-secondary);
  font-size: 13px;
}
</style>
