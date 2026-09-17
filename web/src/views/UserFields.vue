<template>
  <div class="user-fields">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="title">
            <span>用户字段</span>
            <el-tag type="info" size="small">自定义用户字段，数据源变化无需改代码</el-tag>
          </div>
          <div class="header-actions">
            <el-button type="primary" :icon="Plus" :disabled="!canWrite" @click="openCreate">新增字段</el-button>
            <el-button :icon="Refresh" @click="load">刷新</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="字段键作为用户属性（Properties）的键名，数据源的字段映射以此为写入目标；内置字段不可删除，可改为不显示。"
        style="margin-bottom: 12px"
      />

      <el-table v-loading="loading" :data="list" style="width: 100%">
        <el-table-column prop="label" label="显示名" min-width="140">
          <template #default="{ row }">
            {{ row.label }}
            <el-tag v-if="row.builtin" type="info" size="small" effect="plain">内置</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="fieldKey" label="字段键" min-width="140" show-overflow-tooltip />
        <el-table-column prop="fieldType" label="类型" width="110">
          <template #default="{ row }">{{ typeLabel(row.fieldType) }}</template>
        </el-table-column>
        <el-table-column label="列表显示" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.showInList ? 'success' : 'info'" size="small" effect="plain">
              {{ row.showInList ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="表单显示" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.showInForm ? 'success' : 'info'" size="small" effect="plain">
              {{ row.showInForm ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="可编辑" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.editable ? 'success' : 'info'" size="small" effect="plain">
              {{ row.editable ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sortOrder" label="排序" width="80" align="center" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" :disabled="!canWrite" @click="openEdit(row)">编辑</el-button>
            <el-button
              link
              type="danger"
              size="small"
              :disabled="row.builtin || !canWrite"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="formVisible" :title="formIsEdit ? '编辑字段' : '新增字段'" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="字段键" required>
          <el-input v-model="form.fieldKey" :disabled="formIsEdit && form.builtin" placeholder="英文标识，如 jobLevel" />
          <div class="form-tip">作为用户属性键名，创建后不建议修改</div>
        </el-form-item>
        <el-form-item label="显示名" required>
          <el-input v-model="form.label" placeholder="如 职级" />
        </el-form-item>
        <el-form-item label="字段类型">
          <el-select v-model="form.fieldType" style="width: 100%">
            <el-option v-for="t in fieldTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.fieldType === 'select'" label="选项">
          <el-input
            v-model="form.options"
            type="textarea"
            :rows="3"
            placeholder='JSON 数组，如 ["P5","P6","P7"]'
          />
        </el-form-item>
        <el-form-item label="列表显示">
          <el-switch v-model="form.showInList" />
        </el-form-item>
        <el-form-item label="表单显示">
          <el-switch v-model="form.showInForm" />
        </el-form-item>
        <el-form-item label="可编辑">
          <el-switch v-model="form.editable" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sortOrder" :min="0" :max="9999" />
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
import { Plus, Refresh } from '@element-plus/icons-vue'
import { listUserFields, createUserField, updateUserField, deleteUserField } from '@/api/userField'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canWrite = computed(() => authStore.hasPermission('userfield:write'))

const fieldTypes = [
  { value: 'text', label: '单行文本' },
  { value: 'textarea', label: '多行文本' },
  { value: 'number', label: '数字' },
  { value: 'select', label: '下拉选择' },
  { value: 'date', label: '日期' }
]

const loading = ref(false)
const saving = ref(false)
const list = ref([])

const formVisible = ref(false)
const formIsEdit = ref(false)
const emptyForm = () => ({
  fieldKey: '',
  label: '',
  fieldType: 'text',
  options: '',
  showInList: true,
  showInForm: true,
  editable: true,
  sortOrder: 100
})
const form = ref(emptyForm())

const typeLabel = (t) => fieldTypes.find((x) => x.value === t)?.label || t

const load = async () => {
  loading.value = true
  try {
    const res = await listUserFields()
    list.value = res.data || []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.value = emptyForm()
  formIsEdit.value = false
  formVisible.value = true
}

const openEdit = (row) => {
  form.value = { ...row }
  formIsEdit.value = true
  formVisible.value = true
}

const handleSave = async () => {
  if (!form.value.fieldKey || !form.value.label) {
    ElMessage.warning('请填写字段键与显示名')
    return
  }
  saving.value = true
  try {
    if (formIsEdit.value) {
      await updateUserField(form.value.id, form.value)
      ElMessage.success('更新成功')
    } else {
      await createUserField(form.value)
      ElMessage.success('新增成功')
    }
    formVisible.value = false
    load()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除字段「${row.label}（${row.fieldKey}）」吗？`, '提示', { type: 'warning' })
  } catch (e) {
    return
  }
  await deleteUserField(row.id)
  ElMessage.success('已删除')
  load()
}

onMounted(load)
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.form-tip {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.6;
}
</style>
