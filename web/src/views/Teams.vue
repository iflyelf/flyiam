<template>
  <div class="teams-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>团队管理</span>
          <div class="actions">
            <el-input
              v-model="keyword"
              placeholder="搜索团队"
              clearable
              style="width: 200px"
              @keyup.enter="loadData"
              @clear="loadData"
            />
            <el-button type="primary" :icon="Plus" :disabled="!canWrite" @click="openCreate">新建团队</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="团队是权限分配的主体：为团队添加成员并授权角色，成员即获得对应权限"
        style="margin-bottom: 16px"
      />

      <el-table v-loading="loading" :data="teams" stripe border style="width: 100%">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="团队名称" min-width="150" />
        <el-table-column prop="code" label="代码" min-width="120" />
        <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
        <el-table-column label="成员数" width="90" align="center">
          <template #default="{ row }"><el-tag size="small" type="info">{{ row.memberCount }}</el-tag></template>
        </el-table-column>
        <el-table-column label="角色数" width="90" align="center">
          <template #default="{ row }"><el-tag size="small" type="success">{{ row.roleCount }}</el-tag></template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openMembers(row)">成员</el-button>
            <el-button link type="success" size="small" @click="openRoles(row)">角色</el-button>
            <el-button link type="warning" size="small" :disabled="!canWrite" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" :disabled="!canDelete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新建/编辑团队 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑团队' : '新建团队'" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="团队名称" required>
          <el-input v-model="form.name" placeholder="如：运维一组" />
        </el-form-item>
        <el-form-item label="代码">
          <el-input v-model="form.code" placeholder="如：ops1（留空同名称）" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>

    <!-- 成员管理 -->
    <el-dialog v-model="memberVisible" :title="`团队成员 - ${currentTeam?.name || ''}`" width="640px">
      <div class="add-row">
        <el-select
          v-model="selectedUser"
          filterable
          remote
          reserve-keyword
          :remote-method="searchUsers"
          :loading="userSearchLoading"
          placeholder="搜索用户（域账号/姓名）"
          style="flex: 1"
        >
          <el-option
            v-for="u in userOptions"
            :key="u.domainAccount"
            :label="`${u.name}（${u.domainAccount}）`"
            :value="u.domainAccount"
          />
        </el-select>
        <el-button type="primary" :disabled="!selectedUser || !canWrite" @click="handleAddMember">添加</el-button>
      </div>
      <el-table :data="members" v-loading="memberLoading" stripe border style="width: 100%; margin-top: 12px">
        <el-table-column prop="username" label="域账号" min-width="150" />
        <el-table-column prop="displayName" label="姓名" min-width="120" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="danger" size="small" :disabled="!canWrite" @click="handleRemoveMember(row)">
              移除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="memberVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 角色授权 -->
    <el-dialog v-model="roleVisible" :title="`团队角色 - ${currentTeam?.name || ''}`" width="520px">
      <el-checkbox-group v-model="selectedRoleIds" class="role-group">
        <el-checkbox v-for="r in allRoles" :key="r.id" :label="r.id">{{ r.name }}</el-checkbox>
      </el-checkbox-group>
      <span v-if="allRoles.length === 0" class="muted">暂无角色，请先在角色管理中创建</span>
      <template #footer>
        <el-button @click="roleVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" :disabled="!canWrite" @click="handleSaveRoles">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import {
  listTeams,
  createTeam,
  updateTeam,
  deleteTeam,
  listTeamMembers,
  addTeamMember,
  removeTeamMember,
  listTeamRoles,
  setTeamRoles,
  listRoles
} from '@/api/rbac'
import { listUsers } from '@/api/user'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canWrite = computed(() => authStore.hasPermission('team:write'))
const canDelete = computed(() => authStore.hasPermission('team:delete'))

const loading = ref(false)
const saving = ref(false)
const keyword = ref('')
const teams = ref([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const form = ref({ id: null, name: '', code: '', description: '', status: 1 })

const memberVisible = ref(false)
const memberLoading = ref(false)
const members = ref([])
const currentTeam = ref(null)
const selectedUser = ref('')
const userOptions = ref([])
const userSearchLoading = ref(false)

const roleVisible = ref(false)
const allRoles = ref([])
const selectedRoleIds = ref([])

const loadData = async () => {
  loading.value = true
  try {
    const res = await listTeams({ keyword: keyword.value })
    teams.value = res.data?.list || []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.value = { id: null, name: '', code: '', description: '', status: 1 }
  isEdit.value = false
  dialogVisible.value = true
}

const openEdit = (row) => {
  form.value = { ...row }
  isEdit.value = true
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!form.value.name) {
    ElMessage.warning('请填写团队名称')
    return
  }
  saving.value = true
  try {
    if (isEdit.value) {
      await updateTeam(form.value.id, form.value)
      ElMessage.success('团队已更新')
    } else {
      await createTeam(form.value)
      ElMessage.success('团队已创建')
    }
    dialogVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  await ElMessageBox.confirm(`确定删除团队「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteTeam(row.id)
  ElMessage.success('已删除')
  loadData()
}

// ---------------- 成员 ----------------
const openMembers = async (row) => {
  currentTeam.value = row
  memberVisible.value = true
  selectedUser.value = ''
  userOptions.value = []
  await loadMembers(row.id)
}

const loadMembers = async (teamId) => {
  memberLoading.value = true
  try {
    const res = await listTeamMembers(teamId)
    members.value = res.data || []
  } finally {
    memberLoading.value = false
  }
}

const searchUsers = async (query) => {
  if (!query) return
  userSearchLoading.value = true
  try {
    const res = await listUsers({ keyword: query, page: 1, pageSize: 20 })
    userOptions.value = res.data?.list || []
  } finally {
    userSearchLoading.value = false
  }
}

const handleAddMember = async () => {
  const user = userOptions.value.find((u) => u.domainAccount === selectedUser.value)
  await addTeamMember(currentTeam.value.id, {
    username: selectedUser.value,
    displayName: user?.name || selectedUser.value
  })
  ElMessage.success('已添加')
  selectedUser.value = ''
  loadMembers(currentTeam.value.id)
  loadData()
}

const handleRemoveMember = async (row) => {
  await removeTeamMember(currentTeam.value.id, row.username)
  ElMessage.success('已移除')
  loadMembers(currentTeam.value.id)
  loadData()
}

// ---------------- 角色 ----------------
const openRoles = async (row) => {
  currentTeam.value = row
  roleVisible.value = true
  const [roleRes, teamRoleRes] = await Promise.all([listRoles(), listTeamRoles(row.id)])
  allRoles.value = roleRes.data?.list || []
  selectedRoleIds.value = teamRoleRes.data || []
}

const handleSaveRoles = async () => {
  saving.value = true
  try {
    await setTeamRoles(currentTeam.value.id, selectedRoleIds.value)
    ElMessage.success('角色已保存')
    roleVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.actions {
  display: flex;
  gap: 10px;
  align-items: center;
}
.add-row {
  display: flex;
  gap: 10px;
  align-items: center;
}
.role-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.muted {
  color: var(--text-secondary);
}
</style>
