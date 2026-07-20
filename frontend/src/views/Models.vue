<template>
  <n-card title="模型列表" size="small">
    <n-data-table :columns="columns" :data="models" :loading="loading" :bordered="false" size="small" style="width:100%" />
  </n-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { NCard, NDataTable } from 'naive-ui'
import { listModels } from '../api'

const columns = [
  { title: '提供商', key: 'provider', width: 120 },
  { title: '模型', key: 'model', width: 200 },
  { title: '缓存命中价（元/百万token）', key: 'input_cache_hit_price', width: 140, render(r) { return r.input_cache_hit_price.toFixed(4).replace(/0+$/,'').replace(/\.$/,'') }},
  { title: '缓存未命中价（元/百万token）', key: 'input_cache_miss_price', width: 150, render(r) { return r.input_cache_miss_price.toFixed(4).replace(/0+$/,'').replace(/\.$/,'') }},
  { title: '输出价（元/百万token）', key: 'output_price', width: 120, render(r) { return r.output_price.toFixed(4).replace(/0+$/,'').replace(/\.$/,'') }},
  { title: '上下文token', key: 'max_context_tokens', width: 90, render(r) { return r.max_context_tokens ? (r.max_context_tokens/1024).toFixed(1).replace(/0+$/,'').replace(/\.$/,'')+'K' : '-' }},
  { title: '最大输出token', key: 'max_completion_tokens', width: 90, render(r) { return r.max_completion_tokens ? (r.max_completion_tokens/1024).toFixed(1).replace(/0+$/,'').replace(/\.$/,'')+'K' : '-' }},
  { title: '创建时间', key: 'created_at', width: 170, render(r) { return formatTime(r.created_at) }},
]

const models = ref([])
const loading = ref(false)

async function load() {
  loading.value = true
  try { models.value = await listModels() } catch {} finally { loading.value = false }
}

function formatTime(t) {
  if (!t || t.startsWith('0001')) return '-'
  return t.replace('T',' ').substring(0,19)
}

onMounted(() => load())
</script>
