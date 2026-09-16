<template>
  <div class="schedule">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <span>定时任务配置</span>
          <div>
            <el-button :icon="Refresh" @click="load">刷新</el-button>
            <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
          </div>
        </div>
      </template>

      <el-form :model="form" label-width="180px" style="max-width: 720px">
        <el-form-item label="启用定时任务">
          <el-switch v-model="form.enabled" />
          <span class="hint">关闭后不再自动执行，仍可手动同步</span>
        </el-form-item>

        <el-form-item label="执行间隔">
          <el-select v-model="form.interval" style="width: 200px">
            <el-option label="每 30 分钟" value="30m" />
            <el-option label="每 1 小时" value="1h" />
            <el-option label="每 3 小时" value="3h" />
            <el-option label="每 6 小时" value="6h" />
            <el-option label="每 12 小时" value="12h" />
            <el-option label="每天" value="24h" />
          </el-select>
          <span class="hint">支持 Go 时长格式，如 30m / 6h / 24h</span>
        </el-form-item>

        <el-divider content-position="left">执行内容</el-divider>

        <el-form-item label="从数据源同步">
          <el-switch v-model="form.syncDataSource" />
          <span class="hint">拉取人员 API 数据并直接同步到 Casdoor（本系统不存储用户）</span>
        </el-form-item>

        <el-form-item label="删除不存在人员">
          <el-switch v-model="form.deleteMissing" />
          <span class="hint warn">数据源中不存在的人员，直接从 Casdoor 删除（受保护用户除外）</span>
        </el-form-item>

        <el-form-item label="并发批量大小">
          <el-input-number v-model="form.casdoorBatchSize" :min="1" :max="200" />
          <span class="hint">同步并发数，过大可能压垮 Casdoor</span>
        </el-form-item>
      </el-form>

      <el-descriptions :column="1" border style="margin-top: 20px; max-width: 720px">
        <el-descriptions-item label="上次执行时间">{{ formatDateTime(form.lastRunAt) }}</el-descriptions-item>
        <el-descriptions-item label="上次执行结果">
          <el-tag v-if="form.lastRunStatus" :type="form.lastRunStatus === 'success' ? 'success' : 'danger'" size="small">
            {{ form.lastRunStatus === 'success' ? '成功' : '失败' }}
          </el-tag>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item label="上次执行信息">{{ form.lastRunMessage || '-' }}</el-descriptions-item>
      </el-descriptions>

      <el-alert
        style="margin-top: 20px; max-width: 720px"
        type="info"
        :closable="false"
        show-icon
        title="说明：定时任务由后端内置调度器每分钟检查一次，到达间隔即自动执行。修改配置后立即生效，无需重启。"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getSchedule, updateSchedule } from '@/api/schedule'
import { formatDateTime } from '@/utils/time'

const loading = ref(false)
const saving = ref(false)

const form = ref({
  enabled: false,
  interval: '6h',
  syncDataSource: true,
  deleteMissing: true,
  casdoorBatchSize: 10,
  lastRunAt: null,
  lastRunStatus: '',
  lastRunMessage: ''
})

const load = async () => {
  loading.value = true
  try {
    const res = await getSchedule()
    form.value = { ...form.value, ...(res.data || {}) }
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    const res = await updateSchedule(form.value)
    form.value = { ...form.value, ...(res.data || {}) }
    ElMessage.success('配置已保存')
  } finally {
    saving.value = false
  }
}


onMounted(load)
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.hint {
  margin-left: 12px;
  color: var(--text-secondary);
  font-size: 13px;
}
.hint.warn {
  color: var(--warn-color);
}
</style>
