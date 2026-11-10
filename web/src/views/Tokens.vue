<template>
  <div class="tokens">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>API 令牌</span>
            <el-tag type="info" size="small">供其它系统调用本系统接口（如 Consul Manager 同步字段定义）</el-tag>
          </div>
          <div class="actions">
            <el-button
              type="danger"
              plain
              :icon="Delete"
              :disabled="!selection.length"
              @click="handleBatchDelete"
            >
              批量删除{{ selection.length ? `（${selection.length}）` : '' }}
            </el-button>
            <el-button type="primary" :icon="Plus" @click="openCreate">生成令牌</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="令牌仅在生成时显示一次，请立即复制保存；可设置有效期或永久有效，随时可吊销。"
        style="margin-bottom: 12px"
      />

      <el-table
        ref="tableRef"
        :data="list"
        v-loading="loading"
        stripe
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="48" />
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
      <el-input ref="tokenInputRef" v-model="createdToken" readonly>
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
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, CopyDocument, Delete } from '@element-plus/icons-vue'
import { listTokens, createToken, revokeToken, batchDeleteTokens } from '@/api/token'

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const tableRef = ref(null)
const selection = ref([])

const formVisible = ref(false)
const form = ref({ name: '', expiresInDays: 0 })

const resultVisible = ref(false)
const createdToken = ref('')
const tokenInputRef = ref(null)

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
    selection.value = []
    // 数据刷新后清空表格勾选，避免残留已删除行导致批量操作数量不准。
    tableRef.value?.clearSelection()
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
    // 打开弹窗后自动全选令牌，复制失败时用户可直接 Ctrl+C。
    nextTick(() => tokenInputRef.value?.select?.())
    load()
  } finally {
    saving.value = false
  }
}

const onSelectionChange = (rows) => {
  selection.value = rows
}

// copyText 兼容 HTTP 部署：navigator.clipboard 仅在安全上下文（HTTPS / localhost）可用，
// 内网 http:// 访问时必须回退到 textarea + execCommand，否则「复制」按钮形同虚设。
const copyText = async (text) => {
  if (!text) return false
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch (e) {
    // 继续尝试降级方案
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.top = '-1000px'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    ta.setSelectionRange(0, ta.value.length)
    const done = document.execCommand('copy')
    document.body.removeChild(ta)
    return done
  } catch (e) {
    return false
  }
}

const copyToken = async () => {
  const ok = await copyText(createdToken.value)
  if (ok) {
    ElMessage.success('已复制到剪贴板')
    return
  }
  // 复制失败时自动选中文本，方便直接 Ctrl+C。
  tokenInputRef.value?.select?.()
  ElMessage.warning('复制失败，令牌已选中，请按 Ctrl+C 手动复制')
}

const handleBatchDelete = async () => {
  if (!selection.value.length) return
  const names = selection.value.map((r) => r.name || r.tokenPrefix).slice(0, 3).join('、')
  const more = selection.value.length > 3 ? ' 等' : ''
  try {
    await ElMessageBox.confirm(
      `确定删除选中的 ${selection.value.length} 个令牌（${names}${more}）吗？删除后立即失效。`,
      '批量删除',
      { type: 'warning' }
    )
  } catch (e) {
    return
  }
  const res = await batchDeleteTokens(selection.value.map((r) => r.id))
  ElMessage.success(`已删除 ${res.data?.deleted ?? 0} 个令牌`)
  load()
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
.actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.key { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; }
</style>
