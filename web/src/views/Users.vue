<template>
  <div class="users">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>用户管理</span>
            <el-tag type="success" size="small">数据来源：Casdoor</el-tag>
          </div>
          <div class="header-actions">
            <el-button type="primary" :icon="Plus" :disabled="!canWrite" @click="openCreate">新增用户</el-button>
            <el-button
              type="danger"
              :icon="Delete"
              :disabled="selection.length === 0 || !canDelete"
              @click="handleBatchDelete"
            >
              批量删除{{ selection.length ? `(${selection.length})` : '' }}
            </el-button>
            <el-button :icon="Refresh" @click="loadUsers">刷新</el-button>
          </div>
        </div>
      </template>

      <div class="search-bar">
        <el-select v-model="searchField" style="width: 130px">
          <el-option label="域账号" value="name" />
          <el-option label="姓名" value="displayName" />
          <el-option label="邮箱" value="email" />
          <el-option label="手机号" value="phone" />
        </el-select>
        <el-input
          v-model="keyword"
          placeholder="输入关键词"
          :prefix-icon="Search"
          clearable
          style="width: 280px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        <el-button :icon="Refresh" @click="handleReset">重置</el-button>
      </div>

      <el-table
        v-loading="loading"
        :data="users"
        style="width: 100%; margin-top: 16px"
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="46" :selectable="isSelectable" />
        <el-table-column label="用户" min-width="220">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="34" :src="row.avatar || defaultAvatar(row)">
                {{ (row.name || row.domainAccount || '?').slice(0, 1) }}
              </el-avatar>
              <div class="user-meta">
                <div class="user-name">
                  {{ row.name || row.domainAccount }}
                  <el-tag v-if="row.isAdmin" type="danger" size="small">管理员</el-tag>
                  <el-tag v-if="row.isProtected" type="warning" size="small">受保护</el-tag>
                </div>
                <div class="user-sub">{{ row.domainAccount }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="workEmail" label="邮箱" min-width="180" show-overflow-tooltip />
        <el-table-column prop="phone" label="电话" width="130" />
        <!-- 动态字段列：由「用户字段」定义驱动，新增字段无需改代码 -->
        <el-table-column
          v-for="f in listFields"
          :key="f.fieldKey"
          :label="f.label"
          min-width="140"
          show-overflow-tooltip
        >
          <template #default="{ row }">{{ extraValue(row, f.fieldKey) || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small" effect="plain">
              {{ row.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleView(row)">查看</el-button>
            <el-button link type="primary" size="small" :disabled="!canWrite" @click="openEdit(row)">编辑</el-button>
            <el-tooltip content="更多操作" placement="top">
              <el-dropdown trigger="click" @command="(cmd) => handleMore(cmd, row)">
                <el-button link type="primary" size="small" :icon="MoreFilled" />
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="reset" :icon="Key" :disabled="!canWrite">重置密码</el-dropdown-item>
                    <el-dropdown-item command="customPwd" :icon="EditPen" :disabled="!canWrite">自定义密码</el-dropdown-item>
                    <el-dropdown-item command="admin" :icon="UserFilled" :disabled="!canWrite">
                      {{ row.isAdmin ? '取消管理员' : '设为管理员' }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </el-tooltip>
            <el-button
              link
              type="danger"
              size="small"
              :disabled="row.isProtected || !canDelete"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
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
        @size-change="loadUsers"
        @current-change="loadUsers"
      />
    </el-card>

    <!-- 查看详情 -->
    <el-dialog v-model="detailVisible" title="用户详情" width="620px">
      <div class="detail-head" v-if="currentUser">
        <el-avatar :size="56" :src="currentUser.avatar || defaultAvatar(currentUser)">
          {{ (currentUser.name || currentUser.domainAccount || '?').slice(0, 1) }}
        </el-avatar>
        <div class="detail-head-meta">
          <div class="detail-head-name">
            {{ currentUser.name || currentUser.domainAccount }}
            <el-tag v-if="currentUser.isAdmin" type="danger" size="small">管理员</el-tag>
            <el-tag v-if="currentUser.isProtected" type="warning" size="small">受保护</el-tag>
          </div>
          <div class="detail-head-sub">{{ currentUser.domainAccount }}</div>
        </div>
      </div>
      <el-descriptions :column="2" border v-if="currentUser" style="margin-top: 16px">
        <el-descriptions-item label="域账号">
          {{ currentUser.domainAccount }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">{{ currentUser.status === 'active' ? '正常' : '禁用' }}</el-descriptions-item>
        <el-descriptions-item label="姓名">{{ currentUser.name }}</el-descriptions-item>
        <el-descriptions-item label="邮箱">{{ currentUser.workEmail || '-' }}</el-descriptions-item>
        <el-descriptions-item label="电话">{{ currentUser.phone || '-' }}</el-descriptions-item>
        <el-descriptions-item v-for="f in allFields" :key="f.fieldKey" :label="f.label">
          {{ extraValue(currentUser, f.fieldKey) || '-' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 新增/编辑 -->
    <el-dialog v-model="formVisible" :title="formIsEdit ? '编辑用户' : '新增用户'" width="620px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="域账号" required>
          <el-input v-model="form.domainAccount" :disabled="formIsEdit" placeholder="唯一标识，如 zhangsan" />
        </el-form-item>
        <el-form-item label="姓名" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.workEmail" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="form.phone" />
        </el-form-item>
        <!-- 动态字段表单项：由「用户字段」定义驱动（含零级部门等内置字段） -->
        <el-form-item v-for="f in formFields" :key="f.fieldKey" :label="f.label">
          <el-input v-if="f.fieldType === 'text'" v-model="form.extra[f.fieldKey]" />
          <el-input
            v-else-if="f.fieldType === 'textarea'"
            v-model="form.extra[f.fieldKey]"
            type="textarea"
            :rows="2"
          />
          <el-input-number
            v-else-if="f.fieldType === 'number'"
            v-model="form.extra[f.fieldKey]"
            :controls="false"
            style="width: 100%"
          />
          <el-select
            v-else-if="f.fieldType === 'select'"
            v-model="form.extra[f.fieldKey]"
            style="width: 100%"
            clearable
          >
            <el-option v-for="opt in parseOptions(f.options)" :key="opt" :label="opt" :value="opt" />
          </el-select>
          <el-date-picker
            v-else-if="f.fieldType === 'date'"
            v-model="form.extra[f.fieldKey]"
            type="date"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
          <el-input v-else v-model="form.extra[f.fieldKey]" />
        </el-form-item>
        <el-form-item v-if="!formIsEdit" label="初始密码">
          <el-input v-model="form.password" type="password" show-password placeholder="留空使用系统默认密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus, Delete, MoreFilled, Key, EditPen, UserFilled } from '@element-plus/icons-vue'
import {
  listUsers,
  createUser,
  updateUser,
  deleteUser,
  batchDeleteUsers,
  resetPassword,
  setUserAdmin
} from '@/api/user'
import { listUserFields } from '@/api/userField'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canWrite = computed(() => authStore.hasPermission('user:write'))
const canDelete = computed(() => authStore.hasPermission('user:delete'))

// 用户字段定义（页面可配置）：驱动列表列与表单动态渲染
const allFields = ref([])
const listFields = computed(() => allFields.value.filter((f) => f.showInList))
const formFields = computed(() => allFields.value.filter((f) => f.showInForm && f.editable))

// extraValue 读取用户扩展字段值（extar 优先，兼容旧版显式字段）
const extraValue = (row, key) => {
  if (!row) return ''
  const v = row.extra?.[key]
  if (v !== undefined && v !== null && v !== '') return v
  return row[key] ?? ''
}

// parseOptions 解析下拉选项（JSON 数组字符串）
const parseOptions = (options) => {
  if (!options) return []
  try {
    const arr = JSON.parse(options)
    return Array.isArray(arr) ? arr : []
  } catch (e) {
    return String(options)
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  }
}

const loadFields = async () => {
  try {
    const res = await listUserFields()
    allFields.value = res.data || []
  } catch (e) {
    allFields.value = []
  }
}

const loading = ref(false)
const saving = ref(false)
const users = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const searchField = ref('name')
const selection = ref([])

const detailVisible = ref(false)
const currentUser = ref(null)

const formVisible = ref(false)
const formIsEdit = ref(false)
const emptyForm = () => ({
  domainAccount: '',
  name: '',
  workEmail: '',
  phone: '',
  password: '',
  extra: {}
})
const form = ref(emptyForm())

const isSelectable = (row) => !row.isProtected

// defaultAvatar 无头像时用姓名首字生成占位（数据 URI，不依赖外部服务）
const defaultAvatar = (row) => {
  const ch = (row?.name || row?.domainAccount || '?').trim().slice(0, 1).toUpperCase()
  const colors = ['#c8865a', '#2f6fed', '#83a76f', '#d9a94c', '#d9776c', '#8e7cc3']
  let hash = 0
  const key = row?.domainAccount || ch
  for (let i = 0; i < key.length; i++) hash = (hash * 31 + key.charCodeAt(i)) % colors.length
  const bg = colors[hash]
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80"><rect width="80" height="80" rx="40" fill="${bg}"/><text x="50%" y="54%" font-size="34" fill="#fff" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif">${ch}</text></svg>`
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await listUsers({
      field: searchField.value,
      keyword: keyword.value,
      page: page.value,
      pageSize: pageSize.value
    })
    const data = res.data || {}
    users.value = data.list || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  page.value = 1
  loadUsers()
}

const handleReset = () => {
  keyword.value = ''
  searchField.value = 'name'
  page.value = 1
  loadUsers()
}

const onSelectionChange = (rows) => {
  selection.value = rows
}

const handleView = (row) => {
  currentUser.value = row
  detailVisible.value = true
}

// 更多操作：重置密码 / 自定义密码 / 管理员标记
const handleMore = async (cmd, row) => {
  try {
    if (cmd === 'reset') {
      await ElMessageBox.confirm(`确定将「${row.name}」的密码重置为系统默认密码吗？`, '提示', { type: 'warning' })
      const res = await resetPassword(row.domainAccount)
      ElMessage.success(`已重置为：${res.data?.password || '默认密码'}`)
    } else if (cmd === 'customPwd') {
      const { value } = await ElMessageBox.prompt(`为「${row.name}」设置新密码`, '自定义密码', {
        inputType: 'password',
        inputPlaceholder: '请输入新密码',
        inputValidator: (v) => (v && v.length >= 6 ? true : '密码至少 6 位')
      })
      await resetPassword(row.domainAccount, value)
      ElMessage.success('密码已设置')
    } else if (cmd === 'admin') {
      const next = !row.isAdmin
      await ElMessageBox.confirm(
        `确定${next ? '将' : '取消'}「${row.name}」的管理员权限吗？`,
        '提示',
        { type: 'warning' }
      )
      await setUserAdmin(row.domainAccount, next)
      ElMessage.success('已更新')
      loadUsers()
    }
  } catch (e) {
    // 用户取消或请求失败，忽略（错误提示已由请求拦截器统一处理）
  }
}

const openCreate = () => {
  form.value = emptyForm()
  formIsEdit.value = false
  formVisible.value = true
}

const openEdit = (row) => {
  form.value = {
    ...emptyForm(),
    domainAccount: row.domainAccount,
    name: row.name,
    workEmail: row.workEmail,
    phone: row.phone,
    extra: { ...(row.extra || {}) }
  }
  formIsEdit.value = true
  formVisible.value = true
}

const handleSave = async () => {
  if (!form.value.domainAccount || !form.value.name) {
    ElMessage.warning('请填写域账号与姓名')
    return
  }
  saving.value = true
  try {
    if (formIsEdit.value) {
      await updateUser(form.value.domainAccount, form.value)
      ElMessage.success('更新成功')
    } else {
      await createUser(form.value)
      ElMessage.success('新增成功')
    }
    formVisible.value = false
    loadUsers()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除用户「${row.name}（${row.domainAccount}）」吗？`, '提示', {
      type: 'warning'
    })
  } catch (e) {
    return
  }
  await deleteUser(row.domainAccount)
  ElMessage.success('已删除')
  loadUsers()
}

const handleBatchDelete = async () => {
  const names = selection.value.map((r) => r.domainAccount)
  try {
    await ElMessageBox.confirm(`确定批量删除选中的 ${names.length} 个用户吗？`, '提示', {
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const res = await batchDeleteUsers(names)
  const data = res.data || {}
  let msg = `已删除 ${data.deleted || 0} 个`
  if (data.skipped?.length) msg += `，跳过受保护 ${data.skipped.length} 个`
  if (data.failed?.length) msg += `，失败 ${data.failed.length} 个`
  ElMessage.success(msg)
  loadUsers()
}

onMounted(() => {
  loadFields()
  loadUsers()
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
}
.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}
.search-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}
.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.user-meta {
  min-width: 0;
}
.user-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: var(--text-color);
}
.user-sub {
  font-size: 12px;
  color: var(--text-secondary);
}
.detail-head {
  display: flex;
  align-items: center;
  gap: 14px;
}
.detail-head-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 17px;
  font-weight: 600;
  color: var(--text-color);
}
.detail-head-sub {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 2px;
}
</style>
