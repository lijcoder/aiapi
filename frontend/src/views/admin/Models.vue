<template>
  <div>
    <n-card title="模型管理" size="small">
      <template #header-extra>
        <n-space align="center">
          <n-input v-model:value="providerKw" placeholder="提供商" size="small" clearable style="width:140px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
          <n-input v-model:value="modelKw" placeholder="模型" size="small" clearable style="width:160px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
          <n-button size="small" @click="resetAndLoad">查询</n-button>
          <n-button size="small" type="primary" @click="openCreate">新增模型</n-button>
        </n-space>
      </template>
      <n-data-table
        :columns="columns"
        :data="models"
        :loading="tableLoading"
        :bordered="false"
        size="small"
        :scroll-x="1140"
        :pagination="pagination"
        :remote="true"
        @update:page="onPage"
        style="width:100%"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showForm" preset="card" :title="formType==='create'?'新增模型':'编辑模型'" style="width:560px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">提供商 provider</div>
          <n-input v-model:value="form.provider" placeholder="如 openai" :disabled="formType==='edit'" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">模型名 model</div>
          <n-input v-model:value="form.model" placeholder="如 gpt-4o-mini" :disabled="formType==='edit'" />
        </div>
        <div style="display:flex;gap:12px">
          <div style="flex:1">
            <div style="font-size:13px;margin-bottom:6px">缓存命中价（元/百万 token）</div>
            <n-input-number v-model:value="form.input_cache_hit_price" :min="0" :step="0.01" style="width:100%" />
          </div>
          <div style="flex:1">
            <div style="font-size:13px;margin-bottom:6px">缓存未命中价</div>
            <n-input-number v-model:value="form.input_cache_miss_price" :min="0" :step="0.01" style="width:100%" />
          </div>
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">输出价（元/百万 token）</div>
          <n-input-number v-model:value="form.output_price" :min="0" :step="0.01" style="width:100%" />
        </div>
        <div style="display:flex;gap:12px">
          <div style="flex:1">
            <div style="font-size:13px;margin-bottom:6px">上下文 token</div>
            <n-input-number v-model:value="form.max_context_tokens" :min="0" :step="1024" style="width:100%" />
          </div>
          <div style="flex:1">
            <div style="font-size:13px;margin-bottom:6px">最大输出 token</div>
            <n-input-number v-model:value="form.max_completion_tokens" :min="0" :step="1024" style="width:100%" />
          </div>
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">支持模态</div>
          <n-checkbox-group v-model:value="modalFlags">
            <n-space>
              <n-checkbox value="text" label="文本" />
              <n-checkbox value="image" label="图像" />
              <n-checkbox value="video" label="视频" />
            </n-space>
          </n-checkbox-group>
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
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NModal, NInput, NInputNumber, NButton, NSpace, NCheckbox, NCheckboxGroup, NTag, useMessage, useDialog } from 'naive-ui'
import { listModelsAdmin, createModel, updateModel, deleteModel } from '../../api'
import { fix4, formatTime } from '../../utils'

const message = useMessage()
const dialog = useDialog()

const models = ref([])
const tableLoading = ref(false)
const providerKw = ref('')
const modelKw = ref('')
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: false })

function onPage(p) { pagination.value.page = p; load() }
function resetAndLoad() { pagination.value.page = 1; load() }

// 新增/编辑
const showForm = ref(false)
const formType = ref('create')
const form = ref(emptyForm())
const formMsg = ref('')
const submitting = ref(false)
const modalFlags = ref(['text'])

function emptyForm() {
  return {
    id: 0,
    provider: '',
    model: '',
    input_cache_hit_price: 0,
    input_cache_miss_price: 0,
    output_price: 0,
    max_context_tokens: 0,
    max_completion_tokens: 0,
  }
}

function modalToFlags(m) {
  const f = []
  if (m.supports_text) f.push('text')
  if (m.supports_image) f.push('image')
  if (m.supports_video) f.push('video')
  return f
}

function flagsToModal(arr) {
  return {
    supports_text: arr.includes('text'),
    supports_image: arr.includes('image'),
    supports_video: arr.includes('video'),
  }
}

function renderModal(r) {
  const tags = []
  if (r.supports_text) tags.push(h(NTag, { size: 'small', type: 'info', bordered: false }, () => '文本'))
  if (r.supports_image) tags.push(h(NTag, { size: 'small', type: 'success', bordered: false }, () => '图像'))
  if (r.supports_video) tags.push(h(NTag, { size: 'small', type: 'warning', bordered: false }, () => '视频'))
  return tags.length ? h('div', { style: 'display:flex;gap:4px;flex-wrap:wrap' }, tags) : '-'
}

const columns = [
  { title: '提供商', key: 'provider', width: 110 },
  { title: '模型', key: 'model', width: 200, ellipsis: { tooltip: true } },
  { title: '缓存命中价', key: 'input_cache_hit_price', width: 110, render(r) { return '¥' + fix4(r.input_cache_hit_price) } },
  { title: '缓存未命中价', key: 'input_cache_miss_price', width: 120, render(r) { return '¥' + fix4(r.input_cache_miss_price) } },
  { title: '输出价', key: 'output_price', width: 100, render(r) { return '¥' + fix4(r.output_price) } },
  { title: '上下文', key: 'max_context_tokens', width: 90, render(r) { return fmtK(r.max_context_tokens) } },
  { title: '最大输出', key: 'max_completion_tokens', width: 90, render(r) { return fmtK(r.max_completion_tokens) } },
  { title: '能力', key: 'modal', width: 140, render: renderModal },
  { title: '创建时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) } },
  { title: '操作', key: 'actions', width: 120, fixed: 'right', render(r) {
    return h(NSpace, { size: 4 }, () => [
      h(NButton, { size: 'small', tertiary: true, type: 'info', onClick: () => openEdit(r) }, () => '编辑'),
      h(NButton, { size: 'small', tertiary: true, type: 'error', onClick: () => onDelete(r) }, () => '删除'),
    ])
  }},
]

function fmtK(n) {
  if (!n) return '-'
  return (n / 1000).toFixed(1).replace(/0+$/, '').replace(/\.$/, '') + 'K'
}

async function load() {
  tableLoading.value = true
  try {
    const res = await listModelsAdmin(providerKw.value, modelKw.value, pagination.value.page, pagination.value.pageSize)
    models.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

function openCreate() {
  formType.value = 'create'
  form.value = emptyForm()
  modalFlags.value = ['text']
  formMsg.value = ''
  showForm.value = true
}

function openEdit(r) {
  formType.value = 'edit'
  form.value = {
    id: r.id,
    provider: r.provider,
    model: r.model,
    input_cache_hit_price: r.input_cache_hit_price,
    input_cache_miss_price: r.input_cache_miss_price,
    output_price: r.output_price,
    max_context_tokens: r.max_context_tokens,
    max_completion_tokens: r.max_completion_tokens,
  }
  modalFlags.value = modalToFlags(r)
  formMsg.value = ''
  showForm.value = true
}

async function doSubmit() {
  formMsg.value = ''
  if (formType.value === 'create') {
    if (!form.value.provider || !form.value.model) {
      formMsg.value = 'provider 和 model 必填'
      return
    }
  }
  submitting.value = true
  try {
    if (formType.value === 'create') {
      await createModel({
        provider: form.value.provider,
        model: form.value.model,
        input_cache_hit_price: form.value.input_cache_hit_price || 0,
        input_cache_miss_price: form.value.input_cache_miss_price || 0,
        output_price: form.value.output_price || 0,
        max_context_tokens: form.value.max_context_tokens || 0,
        max_completion_tokens: form.value.max_completion_tokens || 0,
        ...flagsToModal(modalFlags.value),
      })
      message.success('创建成功')
    } else {
      await updateModel({
        id: form.value.id,
        input_cache_hit_price: form.value.input_cache_hit_price || 0,
        input_cache_miss_price: form.value.input_cache_miss_price || 0,
        output_price: form.value.output_price || 0,
        max_context_tokens: form.value.max_context_tokens || 0,
        max_completion_tokens: form.value.max_completion_tokens || 0,
        ...flagsToModal(modalFlags.value),
      })
      message.success('已保存')
    }
    showForm.value = false
    await load()
  } catch (e) {
    formMsg.value = e.msg || '操作失败'
  } finally { submitting.value = false }
}

function onDelete(r) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除模型「${r.provider}/${r.model}」吗？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () => {
      deleteModel(r.id).then(() => {
        message.success('已删除')
        load()
      }).catch(e => {
        message.error(e.msg || '删除失败')
      })
    }
  })
}

onMounted(() => load())
</script>
