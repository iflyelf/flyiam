<template>
  <div class="tokens">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>API 令牌</span>
            <el-tag type="info" size="small">供其它系统调用本系统接口（如 Consul Manager 同步字段定义）</el-tag>
          </div>
          <el-button type="primary" :icon="Plus" @click="openCreate">生成令牌</el-button>
        </div>
      </template>

      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="令牌仅在生成时显示一次，请立即复制保存；可设置有效期或永久有效，随时可吊销。"
        style="margin-bottom: 12px"
      />

      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="tokenPrefix" label="令牌前缀" min-width="160">
          <template #default="{ row }"><code class="key">{{ row.tokenPrefix }}</code></template>
        </el-table-column>
        <el-table-column label="有效期" min-width="170">
          <template #default="{ row }">
            <el-tag v-if="!row.expiresAt" type="success" size="small" effect="plain">永久</el-tag>
            <span v-else>{{ fmt(row.expiresAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最后使用" min-width="170">
          <template #default="{ row }">{{ row.lastUsedAt ? fmt(row.lastUsedAt) : '-' }}</template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ fmt(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled && !isExpired(row) ? 'success' : 'danger'" size="small" effect="plain">
              {{ row.enabled && !isExpired(row) ? '有效' : '失效' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="center" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="handleRevoke(row)">吊销</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 生成令牌 -->
    <el-dialog v-model="formVisible" title="生成 API 令牌" width="520px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="如 Consul Manager" />
        </el-form-item>
        <el-form-item label="有效期">
          <el-select v-model="form.expiresInDays" style="width: 100%">
            <el-option label="永久有效" :value="0" />
            <el-option label="30 天" :value="30" />
            <el-option label="90 天" :value="90" />
            <el-option label="180 天" :value="180" />
            <el-option label="365 天" :value="365" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleCreate">生成</el-button>
      </template>
    </el-dialog>

    <!-- 令牌展示（仅一次） -->
    <el-dialog v-model="resultVisible" title="令牌已生成" width="560px">
      <el-alert type="error" :closable="false" show-icon
        title="请立即复制保存，关闭后将无法再次查看。" style="margin-bottom: 12px" />
      <el-input v-model="createdToken" readonly>
        <template #append>
          <el-button :icon="CopyDocument" @click="copyToken">复制</el-button>
        </template>
      </el-input>
      <template #footer>
        <el-button type="primary" @click="resultVisible = false">我已保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, CopyDocument } from '@element-plus/icons-vue'
import { listTokens, createToken, revokeToken } from '@/api/token'

const loading = ref(false)
const saving = ref(false)
const list = ref([])

const formVisible = ref(false)
const form = ref({ name: '', expiresInDays: 0 })

const resultVisible = ref(false)
const createdToken = ref('')

const fmt = (v) => {
  if (!v) return '-'
  const d = new Date(v)
  if (isNaN(d.getTime())) return String(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

const isExpired = (row) => row.expiresAt && new Date(row.expiresAt).getTime() < Date.now()

const load = async () => {
  loading.value = true
  try {
    const res = await listTokens()
    list.value = res.data || []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.value = { name: '', expiresInDays: 0 }
  formVisible.value = true
}

const handleCreate = async () => {
  if (!form.value.name || !form.value.name.trim()) {
    ElMessage.warning('请填写令牌名称')
    return
  }
  saving.value = true
  try {
    const res = await createToken(form.value)
    createdToken.value = res.data?.token || ''
    if (!createdToken.value) {
      ElMessage.error('生成失败：未返回令牌')
      return
    }
    formVisible.value = false
    resultVisible.value = true
    load()
  } finally {
    saving.value = false
  }
}

const copyToken = async () => {
  try {
    await navigator.clipboard.writeText(createdToken.value)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.warning('复制失败，请手动选择复制')
  }
}

const handleRevoke = async (row) => {
  try {
    await ElMessageBox.confirm(`确定吊销令牌「${row.name || row.tokenPrefix}」吗？吊销后立即失效。`, '提示', { type: 'warning' })
  } catch (e) {
    return
  }
  await revokeToken(row.id)
  ElMessage.success('已吊销')
  load()
}

onMounted(load)
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; }
.title { display: flex; align-items: center; gap: 10px; font-weight: 600; }
.key { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; }
</style>
