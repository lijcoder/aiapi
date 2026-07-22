<template>
  <n-card title="模型列表" size="small">
    <n-data-table :columns="columns" :data="models" :loading="loading" :bordered="false" size="small" :scroll-x="1100" style="width:100%" />
  </n-card>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NTag } from 'naive-ui'
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

async function load() {
  loading.value = true
  try { models.value = (await listModels()) || [] } catch {} finally { loading.value = false }
}

onMounted(() => load())
</script>
