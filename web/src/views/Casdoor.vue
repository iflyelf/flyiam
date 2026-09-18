<template>
  <div class="casdoor-admin">
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 应用管理 -->
      <el-tab-pane label="应用管理" name="apps">
        <div class="tab-header">
          <span class="hint">可在此配置应用的 Redirect URI（登录回调白名单），修复登录报错</span>
          <el-button type="primary" :icon="Plus" @click="openAppCreate">新增应用</el-button>
        </div>
        <el-table v-loading="loading.apps" :data="apps" stripe style="width: 100%; margin-top: 12px">
          <el-table-column prop="name" label="应用名" width="150" />
          <el-table-column prop="displayName" label="显示名" width="150" />
          <el-table-column prop="organization" label="组织" width="120" />
          <el-table-column prop="clientId" label="Client ID" min-width="180" show-overflow-tooltip />
          <el-table-column label="回调地址" min-width="280">
            <template #default="{ row }">
              <div v-if="(row.redirectUris || []).length" class="uris">
                <el-tag v-for="u in row.redirectUris" :key="u" size="small" class="uri-tag">{{ u }}</el-tag>
              </div>
              <span v-else class="muted">未配置</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openAppEdit(row)">编辑</el-button>
              <el-button link type="danger" size="small" :disabled="row.name === 'app-built-in'" @click="handleAppDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 组织管理 -->
      <el-tab-pane label="组织管理" name="orgs">
        <div class="tab-header">
          <span class="hint">Casdoor 组织用于隔离用户</span>
          <el-button type="primary" :icon="Plus" @click="openOrgCreate">新增组织</el-button>
        </div>
        <el-table v-loading="loading.orgs" :data="orgs" stripe style="width: 100%; margin-top: 12px">
          <el-table-column prop="name" label="组织名" width="180" />
          <el-table-column prop="displayName" label="显示名" min-width="200" />
          <el-table-column prop="owner" label="Owner" width="120" />
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openOrgEdit(row)">编辑</el-button>
              <el-button link type="danger" size="small" :disabled="row.name === 'built-in'" @click="handleOrgDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 受保护用户 -->
      <el-tab-pane label="受保护用户" name="protected">
        <div class="tab-header">
          <span class="hint">受保护用户不会被同步或删除操作清理（admin 始终受保护）</span>
          <div class="inline-form">
            <el-select
              v-model="protectedForm.accounts"
              multiple
              filterable
              remote
              clearable
              collapse-tags
              collapse-tags-tooltip
              reserve-keyword
              :remote-method="remoteSearchUsers"
              :loading="userSelectLoading"
              placeholder="从用户库选择用户（可多选）"
              style="width: 320px"
            >
              <el-option
                v-for="u in userOptions"
                :key="u.domainAccount"
                :label="u.name ? `${u.name}（${u.domainAccount}）` : u.domainAccount"
                :value="u.domainAccount"
              >
                <span>{{ u.name || u.domainAccount }}</span>
                <span class="option-muted">
                  {{ u.domainAccount }}<template v-if="u.workEmail"> · {{ u.workEmail }}</template>
                </span>
              </el-option>
            </el-select>
            <el-input v-model="protectedForm.remark" placeholder="备注（可选，应用于本次新增）" style="width: 220px" />
            <el-button type="primary" :icon="Plus" @click="handleProtectedAdd">新增</el-button>
          </div>
        </div>
        <el-table v-loading="loading.protected" :data="protectedUsers" stripe style="width: 100%; margin-top: 12px">
          <el-table-column prop="domainAccount" label="域账号" width="220" />
          <el-table-column prop="remark" label="备注" min-width="200" />
          <el-table-column prop="createdAt" label="创建时间" width="180">
            <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="danger"
                size="small"
                :disabled="row.domainAccount === 'admin'"
                @click="handleProtectedDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 应用编辑 -->
    <el-dialog v-model="appDialogVisible" :title="appIsEdit ? '编辑应用' : '新增应用'" width="640px">
      <el-form :model="appForm" label-width="120px">
        <el-form-item label="应用名" required>
          <el-input v-model="appForm.name" :disabled="appIsEdit" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="appForm.displayName" />
        </el-form-item>
        <el-form-item label="组织">
          <el-select v-model="appForm.organization" style="width: 100%">
            <el-option v-for="o in orgs" :key="o.name" :label="o.name" :value="o.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="Client ID">
          <el-input v-model="appForm.clientId" placeholder="留空由 Casdoor 自动生成" />
        </el-form-item>
        <el-form-item label="Client Secret">
          <el-input v-model="appForm.clientSecret" placeholder="留空则保持不变" show-password />
        </el-form-item>
        <el-form-item label="启用密码登录">
          <el-switch v-model="appForm.enablePassword" />
        </el-form-item>
        <el-form-item label="回调地址">
          <el-input
            v-model="appForm.redirectUrisText"
            type="textarea"
            :rows="4"
            placeholder="每行一个，如：&#10;http://10.0.88.88:8081/api/auth/callback&#10;http://localhost:8081/api/auth/callback"
          />
          <span class="hint">每行一个回调地址（Redirect URI 白名单）</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="appDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleAppSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 组织编辑 -->
    <el-dialog v-model="orgDialogVisible" :title="orgIsEdit ? '编辑组织' : '新增组织'" width="480px">
      <el-form :model="orgForm" label-width="100px">
        <el-form-item label="组织名" required>
          <el-input v-model="orgForm.name" :disabled="orgIsEdit" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="orgForm.displayName" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="orgDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleOrgSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  listApplications,
  getApplication,
  createApplication,
  updateApplication,
  deleteApplication,
  listOrganizations,
  createOrganization,
  updateOrganization,
  deleteOrganization,
  listProtectedUsers,
  addProtectedUser,
  deleteProtectedUser
} from '@/api/casdoor'
import { searchUsers } from '@/api/user'
import { formatDateTime } from '@/utils/time'

const activeTab = ref('apps')
const loading = ref({ apps: false, orgs: false, protected: false })
const saving = ref(false)

const apps = ref([])
const orgs = ref([])
const protectedUsers = ref([])
const protectedForm = ref({ accounts: [], remark: '' })
// 用户库下拉选项（排除已在受保护列表中的账号）
const userOptions = ref([])
const userSelectLoading = ref(false)

const appDialogVisible = ref(false)
const appIsEdit = ref(false)
const appForm = ref({})
const orgDialogVisible = ref(false)
const orgIsEdit = ref(false)
const orgForm = ref({})

const loadApps = async () => {
  loading.value.apps = true
  try {
    const res = await listApplications()
    apps.value = res.data || []
  } finally {
    loading.value.apps = false
  }
}

const loadOrgs = async () => {
  loading.value.orgs = true
  try {
    const res = await listOrganizations()
    orgs.value = res.data || []
  } finally {
    loading.value.orgs = false
  }
}

const loadProtected = async () => {
  loading.value.protected = true
  try {
    const res = await listProtectedUsers()
    protectedUsers.value = res.data || []
  } finally {
    loading.value.protected = false
  }
  // 受保护列表变化后刷新可选用户（排除已受保护的账号）
  loadUserOptions()
}

// 从用户库加载可选用户（供下拉选择），支持按域账号/姓名搜索
const loadUserOptions = async (keyword = '') => {
  userSelectLoading.value = true
  try {
    const res = await searchUsers(keyword, 50)
    const protectedSet = new Set(protectedUsers.value.map((p) => p.domainAccount))
    userOptions.value = (res.data || []).filter((u) => !protectedSet.has(u.domainAccount))
  } catch (e) {
    // 加载失败时保持原有选项，不阻断页面
  } finally {
    userSelectLoading.value = false
  }
}

// el-select 远程搜索（按域账号/姓名）
const remoteSearchUsers = (keyword) => {
  loadUserOptions(keyword || '')
}

// ---------------- 应用 ----------------
const openAppCreate = () => {
  appForm.value = {
    name: '',
    displayName: '',
    organization: orgs.value[0]?.name || '',
    clientId: '',
    clientSecret: '',
    enablePassword: true,
    redirectUrisText: ''
  }
  appIsEdit.value = false
  appDialogVisible.value = true
}

const openAppEdit = async (row) => {
  const res = await getApplication(row.name)
  const app = res.data || row
  appForm.value = {
    name: app.name,
    displayName: app.displayName,
    organization: app.organization,
    clientId: app.clientId,
    clientSecret: '',
    enablePassword: !!app.enablePassword,
    redirectUrisText: (app.redirectUris || []).join('\n')
  }
  appIsEdit.value = true
  appDialogVisible.value = true
}

const handleAppSave = async () => {
  if (!appForm.value.name) {
    ElMessage.warning('请填写应用名')
    return
  }
  const payload = {
    name: appForm.value.name,
    displayName: appForm.value.displayName,
    organization: appForm.value.organization,
    clientId: appForm.value.clientId,
    clientSecret: appForm.value.clientSecret,
    enablePassword: appForm.value.enablePassword,
    redirectUris: (appForm.value.redirectUrisText || '')
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
  }
  saving.value = true
  try {
    if (appIsEdit.value) {
      await updateApplication(appForm.value.name, payload)
      ElMessage.success('应用已更新')
    } else {
      await createApplication(payload)
      ElMessage.success('应用已创建')
    }
    appDialogVisible.value = false
    loadApps()
  } finally {
    saving.value = false
  }
}

const handleAppDelete = async (row) => {
  await ElMessageBox.confirm(`确定删除应用「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteApplication(row.name)
  ElMessage.success('已删除')
  loadApps()
}

// ---------------- 组织 ----------------
const openOrgCreate = () => {
  orgForm.value = { name: '', displayName: '' }
  orgIsEdit.value = false
  orgDialogVisible.value = true
}

const openOrgEdit = (row) => {
  orgForm.value = { name: row.name, displayName: row.displayName }
  orgIsEdit.value = true
  orgDialogVisible.value = true
}

const handleOrgSave = async () => {
  if (!orgForm.value.name) {
    ElMessage.warning('请填写组织名')
    return
  }
  saving.value = true
  try {
    if (orgIsEdit.value) {
      await updateOrganization(orgForm.value.name, { displayName: orgForm.value.displayName })
      ElMessage.success('组织已更新')
    } else {
      await createOrganization(orgForm.value)
      ElMessage.success('组织已创建')
    }
    orgDialogVisible.value = false
    loadOrgs()
  } finally {
    saving.value = false
  }
}

const handleOrgDelete = async (row) => {
  await ElMessageBox.confirm(`确定删除组织「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteOrganization(row.name)
  ElMessage.success('已删除')
  loadOrgs()
}

// ---------------- 受保护用户 ----------------
const handleProtectedAdd = async () => {
  const accounts = protectedForm.value.accounts || []
  if (accounts.length === 0) {
    ElMessage.warning('请从用户库选择用户')
    return
  }
  try {
    const res = await addProtectedUser({ accounts, remark: protectedForm.value.remark })
    const added = res.data?.added ?? accounts.length
    ElMessage.success(`已新增 ${added} 个受保护用户`)
    protectedForm.value = { accounts: [], remark: '' }
    loadProtected()
  } catch (e) {
    // 错误提示由请求拦截器统一处理
  }
}

const handleProtectedDelete = async (row) => {
  await ElMessageBox.confirm(`确定将「${row.domainAccount}」移出受保护列表吗？`, '提示', { type: 'warning' })
  await deleteProtectedUser(row.domainAccount)
  ElMessage.success('已删除')
  loadProtected()
}



onMounted(() => {
  loadApps()
  loadOrgs()
  loadProtected()
})
</script>

<style scoped>
.tab-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.inline-form {
  display: flex;
  gap: 8px;
  align-items: center;
}
.hint {
  color: var(--text-secondary);
  font-size: 13px;
}
.option-muted {
  float: right;
  margin-left: 16px;
  color: var(--text-secondary);
  font-size: 12px;
}
.muted {
  color: var(--text-secondary);
}
.uris {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.uri-tag {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
