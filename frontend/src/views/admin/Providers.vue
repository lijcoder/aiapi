<template>
  <div>
    <n-card title="提供商管理" size="small">
      <template #header-extra>
        <n-button size="small" type="primary" @click="openCreate">新增提供商</n-button>
      </template>
      <n-data-table
        :columns="columns"
        :data="providers"
        :loading="tableLoading"
        :bordered="false"
        size="small"
        :scroll-x="600"
        :pagination="pagination"
        :remote="true"
        @update:page="onPage"
        style="width:100%"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showForm" preset="card" :title="formType==='create'?'新增提供商':'编辑提供商'" style="width:560px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">标识 type</div>
          <n-input v-model:value="form.type" placeholder="如 openai" :disabled="formType==='edit'" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">域名 domain</div>
          <n-input v-model:value="form.domain" placeholder="https://api.openai.com" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">请求头 headers（JSON）</div>
          <n-input
            v-model:value="form.headersJSON"
            type="textarea"
            :rows="6"
            placeholder='{"Authorization":["Bearer sk-xxx"]}'
          />
          <div style="font-size:12px;color:#909399;margin-top:4px">标准 JSON 格式，key 为 header 名，value 为字符串数组</div>
        </div>
      </div>
      <p v-if="formMsg" style="color:#d03050;font-size:13px;margin-top:8px">{{ formMsg }}</p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showForm=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doSubmit">确认</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 查看弹窗 -->
    <n-modal v-model:show="showView" preset="card" title="提供商配置详情" style="width:560px">
      <div v-if="viewItem" style="display:flex;flex-direction:column;gap:16px">
        <div>
          <div style="font-size:13px;color:#909399;margin-bottom:4px">标识</div>
          <div style="font-weight:600">{{ viewItem.type }}</div>
        </div>
        <div>
          <div style="font-size:13px;color:#909399;margin-bottom:4px">域名</div>
          <div>{{ viewItem.config.domain || '-' }}</div>
        </div>
        <div>
          <div style="font-size:13px;color:#909399;margin-bottom:8px">请求头</div>
          <n-data-table
            v-if="headerRows.length"
            :columns="headerColumns"
            :data="headerRows"
            :bordered="false"
            size="small"
          />
          <div v-else style="color:#909399;font-size:13px">无</div>
        </div>
        <div>
          <div style="font-size:13px;color:#909399;margin-bottom:4px">状态</div>
          <n-tag size="small" :type="viewItem.enabled?'success':'error'">{{ viewItem.enabled ? '启用' : '禁用' }}</n-tag>
        </div>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showView=false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NModal, NInput, NButton, NSpace, NTag, useMessage, useDialog } from 'naive-ui'
import { listProviders, createProvider, updateProvider, toggleProvider } from '../../api'
import { formatTime } from '../../utils'

const message = useMessage()
const dialog = useDialog()

const providers = ref([])
const tableLoading = ref(false)
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: false })

function onPage(p) { pagination.value.page = p; load() }

// 新增/编辑
const showForm = ref(false)
const formType = ref('create')
const form = ref({ type: '', domain: '', headersJSON: '' })
const formMsg = ref('')

// 查看
const showView = ref(false)
const viewItem = ref(null)
const headerRows = ref([])
const headerColumns = [
  { title: 'Header', key: 'key', width: 180 },
  { title: 'Value', key: 'value' },
]

const submitting = ref(false)

const columns = [
  { title: '标识', key: 'type', width: 180, ellipsis: { tooltip: true } },
  { title: '状态', key: 'enabled', width: 100, render(r) {
    return r.enabled ? h(NTag, { size:'small', type:'success' }, { default: () => '启用' }) : h(NTag, { size:'small', type:'error' }, { default: () => '禁用' })
  }},
  { title: '创建时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) } },
  { title: '操作', key: 'actions', width: 220, fixed: 'right', render(r) {
    return h(NSpace, { size: 6 }, () => [
      h(NButton, { size: 'small', tertiary: true, onClick: () => openView(r) }, () => '查看'),
      h(NButton, { size: 'small', tertiary: true, type: 'info', onClick: () => openEdit(r) }, () => '编辑'),
      h(NButton, {
        size: 'small', tertiary: true, type: r.enabled ? 'warning' : 'success',
        onClick: () => onToggle(r)
      }, () => r.enabled ? '禁用' : '启用'),
    ])
  }},
]

async function load() {
  tableLoading.value = true
  try {
    const res = await listProviders(pagination.value.page, pagination.value.pageSize)
    providers.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

function openCreate() {
  formType.value = 'create'
  form.value = { type: '', domain: '', headersJSON: '' }
  formMsg.value = ''
  showForm.value = true
}

function openEdit(r) {
  formType.value = 'edit'
  form.value = {
    type: r.type,
    domain: r.config?.domain || '',
    headersJSON: r.config?.headers ? JSON.stringify(r.config.headers, null, 2) : ''
  }
  formMsg.value = ''
  showForm.value = true
}

async function doSubmit() {
  formMsg.value = ''
  if (!form.value.type) { formMsg.value = 'type 不能为空'; return }
  if (!form.value.domain) { formMsg.value = 'domain 不能为空'; return }

  let headers = {}
  if (form.value.headersJSON.trim()) {
    try {
      headers = JSON.parse(form.value.headersJSON)
    } catch {
      formMsg.value = 'headers 不是合法的 JSON'
      return
    }
  }

  submitting.value = true
  try {
    if (formType.value === 'create') {
      await createProvider(form.value.type, form.value.domain, headers)
      message.success('创建成功')
    } else {
      await updateProvider(form.value.type, form.value.domain, headers)
      message.success('已保存')
    }
    showForm.value = false
    await load()
  } catch (e) {
    formMsg.value = e.msg || '操作失败'
  } finally { submitting.value = false }
}

async function onToggle(r) {
  if (r.enabled) {
    dialog.warning({
      title: '确认禁用',
      content: `确定禁用提供商「${r.type}」吗？禁用后该上游将无法被调用。`,
      positiveText: '禁用',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await toggleProvider(r.type)
          await load()
          message.success('已禁用')
        } catch (e) {
          message.error(e.msg || '操作失败')
        }
      }
    })
    return
  }
  try {
    await toggleProvider(r.type)
    await load()
    message.success('已启用')
  } catch (e) {
    message.error(e.msg || '操作失败')
  }
}

function openView(r) {
  viewItem.value = r
  const headers = r.config?.headers || {}
  headerRows.value = Object.keys(headers).map(k => ({
    key: k,
    value: Array.isArray(headers[k]) ? headers[k].join(', ') : String(headers[k])
  }))
  showView.value = true
}

onMounted(() => load())
</script>
