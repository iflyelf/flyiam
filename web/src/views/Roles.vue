<template>
  <div class="roles-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>角色管理</span>
          <div class="actions">
            <el-input
              v-model="keyword"
              placeholder="搜索角色"
              clearable
              style="width: 200px"
              @keyup.enter="loadData"
              @clear="loadData"
            />
            <el-button type="primary" :icon="Plus" :disabled="!canWrite" @click="openCreate">新建角色</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="角色是权限的集合，可被团队引用；用户通过加入团队获得对应权限"
        style="margin-bottom: 16px"
      />

      <el-table v-loading="loading" :data="roles" stripe border style="width: 100%">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="角色名称" min-width="150" />
        <el-table-column prop="code" label="代码" min-width="120" />
        <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
        <el-table-column label="权限" min-width="280">
          <template #default="{ row }">
            <el-tag
              v-for="p in row.permissions"
              :key="p"
              size="small"
              style="margin: 0 6px 4px 0"
              :type="permType(p)"
            >
              {{ permLabel(p) }}
            </el-tag>
            <span v-if="!row.permissions || row.permissions.length === 0" class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" :disabled="!canWrite" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" :disabled="!canDelete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑角色' : '新建角色'" width="620px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="角色名称" required>
          <el-input v-model="form.name" placeholder="如：用户管理员" />
        </el-form-item>
        <el-form-item label="代码">
          <el-input v-model="form.code" placeholder="如：user_admin（留空同名称）" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="权限">
          <el-checkbox-group v-model="form.permissions" class="perm-group">
            <el-checkbox v-for="p in allPermissions" :key="p" :label="p">{{ permLabel(p) }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { listRoles, createRole, updateRole, deleteRole } from '@/api/rbac'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canWrite = computed(() => authStore.hasPermission('role:write'))
const canDelete = computed(() => authStore.hasPermission('role:delete'))

const loading = ref(false)
const saving = ref(false)
const keyword = ref('')
const roles = ref([])
const allPermissions = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const form = ref({ id: null, name: '', code: '', description: '', permissions: [] })

const permLabels = {
  'user:read': '用户查看', 'user:write': '用户编辑', 'user:delete': '用户删除',
  'team:read': '团队查看', 'team:write': '团队编辑', 'team:delete': '团队删除',
  'role:read': '角色查看', 'role:write': '角色编辑', 'role:delete': '角色删除',
  'sync:read': '同步查看', 'sync:write': '同步执行',
  'datasource:read': '数据源查看', 'datasource:write': '数据源编辑', 'datasource:delete': '数据源删除',
  'schedule:read': '定时查看', 'schedule:write': '定时编辑',
  'casdoor:read': 'Casdoor查看', 'casdoor:write': 'Casdoor编辑'
}
const permLabel = (p) => permLabels[p] || p
const permType = (p) => {
  if (p.endsWith(':delete')) return 'danger'
  if (p.endsWith(':write')) return 'warning'
  return 'info'
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await listRoles({ keyword: keyword.value })
    const data = res.data || {}
    roles.value = data.list || []
    if (data.allPermissions) allPermissions.value = data.allPermissions
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.value = { id: null, name: '', code: '', description: '', permissions: [] }
  isEdit.value = false
  dialogVisible.value = true
}

const openEdit = (row) => {
  form.value = {
    id: row.id,
    name: row.name,
    code: row.code,
    description: row.description,
    permissions: [...(row.permissions || [])]
  }
  isEdit.value = true
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!form.value.name) {
    ElMessage.warning('请填写角色名称')
    return
  }
  saving.value = true
  try {
    if (isEdit.value) {
      await updateRole(form.value.id, form.value)
      ElMessage.success('角色已更新')
    } else {
      await createRole(form.value)
      ElMessage.success('角色已创建')
    }
    dialogVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  await ElMessageBox.confirm(`确定删除角色「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteRole(row.id)
  ElMessage.success('已删除')
  loadData()
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
.muted {
  color: var(--text-secondary);
}
.perm-group {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 16px;
}
</style>
