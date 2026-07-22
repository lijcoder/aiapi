<template>
  <div style="display:flex;flex-direction:column;gap:16px">
    <!-- 顶部指标 -->
    <n-card size="small">
      <div style="display:flex;flex-wrap:wrap;gap:12px 28px">
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">余额</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px;color:#e53935">¥{{ fix4(balance) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">总请求</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">{{ fNum(summary.request_count) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">总Token</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">{{ fNum(summary.total_tokens) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">总费用</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">¥{{ fix4(summary.total_cost) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">平均费用</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">¥{{ fix4(summary.avg_cost) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">输入Token</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">{{ fNum(summary.input_tokens) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">输出Token</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">{{ fNum(summary.output_tokens) }}</div>
        </div>
        <div style="text-align:center;min-width:80px">
          <div style="font-size:12px;color:#666">缓存命中率</div>
          <div style="font-size:14px;font-weight:600;margin-top:2px">{{ fmtRate(summary.cache_hit_rate) }}</div>
        </div>
      </div>
    </n-card>

    <!-- 筛选区 -->
    <n-card size="small">
      <div style="display:flex;align-items:flex-end;flex-wrap:wrap;gap:12px">
        <div>
          <div style="font-size:13px;margin-bottom:4px">统计粒度</div>
          <n-radio-group v-model:value="query.mode" @update:value="onModeChange">
            <n-radio-button value="day">天</n-radio-button>
            <n-radio-button value="month">月</n-radio-button>
          </n-radio-group>
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:4px">时间范围</div>
          <n-date-picker
            v-model:value="dateRange"
            :type="query.mode === 'month' ? 'monthrange' : 'daterange'"
            placement="bottom-start"
            :style="{width:'250px'}"
            clearable
          />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:4px">API Key</div>
          <n-select v-model:value="query.apiKeyId" :options="filterOpts.apiKeys" placeholder="全部" clearable style="width:150px" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:4px">模型</div>
          <n-select v-model:value="query.model" :options="filterOpts.models" placeholder="全部" clearable filterable style="width:170px" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:4px">提供商</div>
          <n-select v-model:value="query.provider" :options="filterOpts.providers" placeholder="全部" clearable style="width:130px" />
        </div>
        <div style="padding-top:20px">
          <n-button type="primary" :loading="loading" @click="doQuery">查询</n-button>
        </div>
      </div>
    </n-card>

    <!-- 维度分析 -->
    <n-card size="small">
      <template #header>
        <n-radio-group v-model:value="query.groupBy" size="small" @update:value="doQuery">
          <n-radio-button value="">时间趋势</n-radio-button>
          <n-radio-button value="model">按模型</n-radio-button>
          <n-radio-button value="provider">按提供商</n-radio-button>
          <n-radio-button value="api_key">按 Key</n-radio-button>
        </n-radio-group>
      </template>
      <div ref="chartEl" style="width:100%;height:340px;margin-bottom:16px" />
      <div style="display:flex;justify-content:center;margin-bottom:12px">
        <n-radio-group v-model:value="chartMetric" size="small">
          <n-radio-button value="cost">费用</n-radio-button>
          <n-radio-button value="total_tokens">总Token</n-radio-button>
          <n-radio-button value="cached_tokens">缓存命中</n-radio-button>
          <n-radio-button value="cache_hit_rate">缓存命中率</n-radio-button>
        </n-radio-group>
      </div>
      <n-data-table :columns="cols" :data="stats" :loading="loading" :bordered="false" size="small" :scroll-x="1200" style="width:100%" />
    </n-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, h } from 'vue'
import { NCard, NDataTable, NInput, NSelect, NButton, NRadioGroup, NRadioButton, NDatePicker, useMessage } from 'naive-ui'
import { usageStats, usageFilters } from '../api'
import { useUser } from '../stores/user'
import { fix4 } from '../utils'
import { useChart } from '../composables/useChart'
import { buildTimeTrendOption, buildDimensionOption, metricConfig } from '../charts'

const message = useMessage()
const { user, fetchUser } = useUser()

const balance = ref(0)
const loading = ref(false)

const dateRange = ref(null)

const query = reactive({
  mode: 'month',
  apiKeyId: null,
  model: null,
  provider: null,
  groupBy: ''
})

const filterOpts = reactive({ apiKeys: [], models: [], providers: [] })

function initDefaultDates() {
  const now = new Date()
  if (query.mode === 'day') {
    const start = new Date(now)
    start.setDate(start.getDate() - 7)
    dateRange.value = [start.getTime(), now.getTime()]
  } else {
    const start = new Date(now.getFullYear(), now.getMonth(), 1)
    dateRange.value = [start.getTime(), now.getTime()]
  }
}

function fmtTS(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  if (query.mode === 'month') {
    return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0')
  }
  return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0')
}
function onModeChange() { initDefaultDates() }

const stats = ref([])

// 顶部汇总指标
const summary = ref({ request_count: 0, input_tokens: 0, output_tokens: 0, cached_tokens: 0, cache_miss_tokens: 0, reasoning_tokens: 0, total_tokens: 0, cache_hit_rate: 0, total_cost: 0, avg_cost: 0 })

function fmtNum(n) { return n != null ? Number(n).toLocaleString() : '-' }
function fNum(n) { return (n || 0).toLocaleString() }
function fmtCost(n) { return n != null ? '¥' + fix4(n) : '-' }
function fmtRate(n) { return n != null ? (n * 100).toFixed(1) + '%' : '-' }

function labelTitle() {
  switch (query.groupBy) {
    case 'model': return '模型'
    case 'provider': return '提供商'
    case 'api_key': return 'API Key'
    default: return '日期'
  }
}

const cols = computed(() => [
  { title: labelTitle(), key: 'label', width: 180, ellipsis: { tooltip: true }, fixed: 'left',
    render(r) {
      if (query.groupBy !== 'api_key') return r.label
      const deleted = r.key_exists === false
      return h('div', [
        h('div', { style: deleted ? 'color:#d03050;font-weight:500' : 'font-weight:500' }, r.sub_label || '-'),
        h('div', { style: 'font-size:12px;color:#909399' }, r.label)
      ])
    }
  },
  { title: '请求次数', key: 'request_count', width: 100, render(r) { return fmtNum(r.request_count) }, sorter: (a, b) => a.request_count - b.request_count },
  { title: '输入Token', key: 'input_tokens', width: 120, render(r) { return fmtNum(r.input_tokens) }, sorter: (a, b) => a.input_tokens - b.input_tokens },
  { title: '输出Token', key: 'output_tokens', width: 120, render(r) { return fmtNum(r.output_tokens) }, sorter: (a, b) => a.output_tokens - b.output_tokens },
  { title: '缓存命中', key: 'cached_tokens', width: 110, render(r) { return fmtNum(r.cached_tokens) }, sorter: (a, b) => a.cached_tokens - b.cached_tokens },
  { title: '缓存未命中', key: 'cache_miss_tokens', width: 110, render(r) { return fmtNum(r.cache_miss_tokens) }, sorter: (a, b) => a.cache_miss_tokens - b.cache_miss_tokens },
  { title: '推理Token', key: 'reasoning_tokens', width: 110, render(r) { return fmtNum(r.reasoning_tokens) }, sorter: (a, b) => a.reasoning_tokens - b.reasoning_tokens },
  { title: '总Token', key: 'total_tokens', width: 120, render(r) { return fmtNum(r.total_tokens) }, sorter: (a, b) => a.total_tokens - b.total_tokens },
  { title: '缓存命中率', key: 'cache_hit_rate', width: 110, render(r) { return fmtRate(r.cache_hit_rate) }, sorter: (a, b) => a.cache_hit_rate - b.cache_hit_rate },
  { title: '费用', key: 'cost', width: 130, render(r) { return fmtCost(r.cost) }, sorter: (a, b) => a.cost - b.cost }
])

async function doQuery() {
  if (!dateRange.value || dateRange.value.length < 2) {
    message.warning('请选择时间范围')
    return
  }
  loading.value = true
  try {
    const data = await usageStats(
      query.mode, fmtTS(dateRange.value[0]), fmtTS(dateRange.value[1]),
      query.apiKeyId, query.model, query.provider, query.groupBy
    )
    summary.value = data.summary || {}
    stats.value = data.rows || []
    await fetchUser()
    balance.value = user.value?.budget || 0
  } catch (e) {
    message.error(e.msg || '查询失败')
  } finally {
    loading.value = false
  }
}

async function loadFilters() {
  try {
    const data = await usageFilters()
    filterOpts.apiKeys = (data.api_keys || []).map(k => ({ label: k.name ? `${k.name} (${k.key})` : k.key, value: k.id }))
    filterOpts.models = (data.models || []).map(m => ({ label: m, value: m }))
    filterOpts.providers = (data.providers || []).map(p => ({ label: p, value: p }))
  } catch { /* ignore */ }
}

const chartEl = ref(null)
const chartMetric = ref('cost')

useChart(chartEl, [stats, chartMetric], (chart) => {
  const rows = stats.value
  if (!rows || !rows.length) { chart.clear(); return }
  if (!query.groupBy) {
    chart.setOption(buildTimeTrendOption(rows, chartMetric.value), true)
  } else {
    chart.setOption(buildDimensionOption(rows, chartMetric.value, summary.value), true)
  }
})

onMounted(() => {
  initDefaultDates()
  loadFilters()
  doQuery()
})
</script>
