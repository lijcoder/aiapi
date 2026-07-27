<template>
  <n-card title="模型列表" size="small">
    <template #header-extra>
      <n-space align="center">
        <n-input v-model:value="providerKw" placeholder="提供商" size="small" clearable style="width:140px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
        <n-input v-model:value="modelKw" placeholder="模型" size="small" clearable style="width:160px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
        <n-button size="small" @click="resetAndLoad">查询</n-button>
      </n-space>
    </template>
    <n-data-table :columns="columns" :data="models" :loading="loading" :bordered="false" size="small" :scroll-x="1100" :pagination="pagination" :remote="true" @update:page="onPage" style="width:100%" />
  </n-card>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NInput, NButton, NSpace, NTag } from 'naive-ui'
import { listModels } from '../api'
import { fix4, formatTime } from '../utils'

const columns = [
  { title: '提供商', key: 'provider', width: 110 },
  { title: '模型', key: 'model', width: 200, ellipsis: { tooltip: true } },
  { title: '缓存命中价', key: 'input_cache_hit_price', width: 110, render(r) { return '¥' + fix4(r.input_cache_hit_price) }},
  { title: '缓存未命中价', key: 'input_cache_miss_price', width: 120, render(r) { return '¥' + fix4(r.input_cache_miss_price) }},
  { title: '输出价', key: 'output_price', width: 100, render(r) { return '¥' + fix4(r.output_price) }},
  { title: '上下文', key: 'max_context_tokens', width: 80, render(r) { return r.max_context_tokens ? (r.max_context_tokens/1000).toFixed(1).replace(/0+$/,'').replace(/\.$/,'')+'K' : '-' }},
  { title: '最大输出', key: 'max_completion_tokens', width: 80, render(r) { return r.max_completion_tokens ? (r.max_completion_tokens/1000).toFixed(1).replace(/0+$/,'').replace(/\.$/,'')+'K' : '-' }},
  { title: '能力', key: 'modal', width: 140, render(r) {
    const tags = []
    if (r.supports_text) tags.push(h(NTag, { size: 'small', type: 'info', bordered: false }, () => '文本'))
    if (r.supports_image) tags.push(h(NTag, { size: 'small', type: 'success', bordered: false }, () => '图像'))
    if (r.supports_video) tags.push(h(NTag, { size: 'small', type: 'warning', bordered: false }, () => '视频'))
    return tags.length ? h('div', { style: 'display:flex;gap:4px;flex-wrap:wrap' }, tags) : '-'
  }},
  { title: '创建时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) }},
]

const models = ref([])
const loading = ref(false)
const providerKw = ref('')
const modelKw = ref('')
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: false })

function onPage(p) { pagination.value.page = p; load() }
function resetAndLoad() { pagination.value.page = 1; load() }

async function load() {
  loading.value = true
  try {
    const res = await listModels(providerKw.value, modelKw.value, pagination.value.page, pagination.value.pageSize)
    models.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch {} finally { loading.value = false }
}

onMounted(() => load())
</script>
