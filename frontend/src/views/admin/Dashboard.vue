<template>
  <div style="display:flex;flex-direction:column;gap:16px">
    <!-- 指标卡 -->
    <n-card size="small">
      <div style="display:flex;align-items:flex-start;gap:24px;flex-wrap:wrap">
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">用户数</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px;color:#1677ff">{{ summary.user_count }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">API Key 数</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px;color:#1677ff">{{ summary.api_key_count }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">今日请求</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px;color:#18a058">{{ summary.today_requests }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">今日费用</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px;color:#18a058">¥{{ fix4(summary.today_cost) }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">今日输入Token</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px">{{ fNum(summary.today_input) }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">今日输出Token</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px">{{ fNum(summary.today_output) }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">今日总Token</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px">{{ fNum(summary.today_total) }}</div>
        </div>
        <div style="text-align:center;min-width:100px">
          <div style="font-size:12px;color:#909399">今日缓存命中率</div>
          <div style="font-size:22px;font-weight:600;margin-top:4px">{{ fmtRate(summary.today_cache_hit) }}</div>
        </div>
      </div>
    </n-card>

    <!-- 7天费用趋势 -->
    <n-card size="small">
      <template #header>
        <div style="display:flex;align-items:center;gap:16px">
          <span>近 7 天趋势</span>
          <n-radio-group v-model:value="metric" size="small">
            <n-radio-button value="cost">费用</n-radio-button>
            <n-radio-button value="request_count">请求数</n-radio-button>
            <n-radio-button value="input_tokens">输入Token</n-radio-button>
            <n-radio-button value="output_tokens">输出Token</n-radio-button>
            <n-radio-button value="total_tokens">总Token</n-radio-button>
            <n-radio-button value="cache_hit_rate">缓存命中率</n-radio-button>
          </n-radio-group>
        </div>
      </template>
      <div ref="chartEl" style="width:100%;height:320px" />
    </n-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { NCard, NRadioGroup, NRadioButton, useMessage } from 'naive-ui'
import { dashboardOverview } from '../../api'
import { fix4 } from '../../utils'
import { useChart } from '../../composables/useChart'

const message = useMessage()
const summary = ref({ user_count: 0, api_key_count: 0, today_requests: 0, today_cost: 0, today_input: 0, today_output: 0, today_total: 0, today_cache_hit: 0 })
function fNum(n) { return (n || 0).toLocaleString() }
function fmtRate(n) { return n != null ? (n * 100).toFixed(1) + '%' : '-' }
const trend = ref([])
const chartEl = ref(null)
const metric = ref('cost')

const metricMeta = {
  cost: { name: '费用', color: '#18a058', format: v => '¥' + Number(v).toFixed(2), area: true },
  input_tokens: { name: '输入Token', color: '#1677ff', format: v => Number(v).toLocaleString() },
  output_tokens: { name: '输出Token', color: '#d03050', format: v => Number(v).toLocaleString() },
  total_tokens: { name: '总Token', color: '#f0a020', format: v => Number(v).toLocaleString() },
  request_count: { name: '请求数', color: '#2080f0', format: v => Number(v).toLocaleString() },
  cache_hit_rate: { name: '缓存命中率', color: '#9254de', format: v => (Number(v) * 100).toFixed(1) + '%', area: true }
}

useChart(chartEl, [trend, metric], (chart) => {
  const rows = trend.value
  if (!rows || !rows.length) { chart.clear(); return }
  const meta = metricMeta[metric.value]
  chart.setOption({
    tooltip: { trigger: 'axis', valueFormatter: v => meta.format(v) },
    grid: { left: 60, right: 30, top: 30, bottom: 30 },
    xAxis: {
      type: 'category',
      data: rows.map(r => r.label),
      axisLabel: { fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      name: meta.name,
      axisLabel: { fontSize: 11, formatter: v => meta.format(v) }
    },
    series: [{
      name: meta.name,
      type: 'line',
      smooth: true,
      data: rows.map(r => Number(r[metric.value]) || 0),
      itemStyle: { color: meta.color },
      areaStyle: meta.area ? { opacity: 0.1 } : undefined
    }]
  }, true)
})

async function load() {
  try {
    const data = await dashboardOverview()
    summary.value = data.summary || {}
    trend.value = data.trend || []
  } catch (e) {
    message.error(e.msg || '加载失败')
  }
}

onMounted(() => load())
</script>
