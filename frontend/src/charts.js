import * as echarts from 'echarts'
import { fix4 } from './utils'

// ===== 指标配置 =====

export const metricConfig = {
  cost: { label: '费用', prefix: '¥', format: v => fix4(v) },
  total_tokens: { label: '总Token', prefix: '', format: v => Number(v).toLocaleString() },
  cached_tokens: { label: '缓存命中', prefix: '', format: v => Number(v).toLocaleString() },
  cache_hit_rate: { label: '缓存命中率', prefix: '', format: v => (v * 100).toFixed(1) + '%', isRate: true }
}

// ===== 通用 tooltip formatter =====

function axisFormatter(ps) {
  let s = ps[0].axisValue + '<br/>'
  ps.forEach(p => {
    const v = p.seriesName === '费用' ? '¥' + fix4(p.value) : Number(p.value).toLocaleString()
    s += p.marker + p.seriesName + ': ' + v + '<br/>'
  })
  return s
}

function pieFormatter(mc) {
  return p => {
    const fullName = p.name
    return fullName + '<br/>' + mc.label + ': ' + mc.prefix + mc.format(p.value) + '<br/>占比: ' + p.percent + '%'
  }
}

// ===== 渐变面积样式 =====

function gradientArea(r, g, b) {
  return new echarts.graphic.LinearGradient(0, 0, 0, 1, [
    { offset: 0, color: `rgba(${r},${g},${b},0.25)` },
    { offset: 1, color: `rgba(${r},${g},${b},0.02)` }
  ])
}

// ===== 时间趋势 =====

/**
 * 构建时间趋势图表 option
 * @param {Array} rows — 后端返回的统计行
 * @param {string} metric — 'cost' | 'total_tokens' | 'cached_tokens'
 */
export function buildTimeTrendOption(rows, metric) {
  const labels = rows.map(r => r.label)

  if (metric === 'cost') {
    return {
      tooltip: { trigger: 'axis', formatter: ps => ps[0].axisValue + '<br/>' + ps[0].marker + '费用: ¥' + fix4(ps[0].value) },
      grid: { top: 20, left: 64, right: 16, bottom: 28 },
      xAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11 }, boundaryGap: false },
      yAxis: { type: 'value', axisLabel: { fontSize: 11, formatter: v => '¥' + (v >= 1 ? v.toFixed(0) : fix4(v)) } },
      series: [
        { name: '费用', type: 'line', smooth: true, symbol: 'circle', symbolSize: 6, data: rows.map(r => r.cost), lineStyle: { width: 2.5, color: '#e0a040' }, itemStyle: { color: '#e0a040' }, areaStyle: { color: gradientArea(224, 160, 64) } }
      ]
    }
  }

  if (metric === 'cached_tokens') {
    return {
      tooltip: { trigger: 'axis', formatter: axisFormatter },
      legend: { top: 0 },
      grid: { top: 36, left: 64, right: 16, bottom: 28 },
      xAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11 }, boundaryGap: false },
      yAxis: { type: 'value', axisLabel: { fontSize: 11, formatter: v => v >= 1000 ? (v / 1000).toFixed(0) + 'k' : v } },
      series: [
        { name: '缓存命中', type: 'line', smooth: true, data: rows.map(r => r.cached_tokens), lineStyle: { width: 2 }, itemStyle: { color: '#95de64' }, areaStyle: { color: gradientArea(149, 222, 100) } },
        { name: '缓存未命中', type: 'line', smooth: true, data: rows.map(r => r.cache_miss_tokens), lineStyle: { width: 2 }, itemStyle: { color: '#ff7875' }, areaStyle: { color: gradientArea(255, 120, 117) } }
      ]
    }
  }

  if (metric === 'cache_hit_rate') {
    return {
      tooltip: { trigger: 'axis', formatter: ps => ps[0].axisValue + '<br/>' + ps[0].marker + '缓存命中率: ' + (ps[0].value * 100).toFixed(1) + '%' },
      grid: { top: 20, left: 64, right: 16, bottom: 28 },
      xAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11 }, boundaryGap: false },
      yAxis: { type: 'value', min: 0, max: 1, axisLabel: { fontSize: 11, formatter: v => (v * 100).toFixed(0) + '%' } },
      series: [
        { name: '缓存命中率', type: 'line', smooth: true, symbol: 'circle', symbolSize: 6, data: rows.map(r => r.cache_hit_rate), lineStyle: { width: 2.5, color: '#73d13d' }, itemStyle: { color: '#73d13d' }, areaStyle: { color: gradientArea(115, 209, 61) } }
      ]
    }
  }

  // total_tokens：多线 + 费用双轴
  return {
    tooltip: { trigger: 'axis', formatter: axisFormatter },
    legend: { top: 0 },
    grid: { top: 40, left: 60, right: 64, bottom: 28 },
    xAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11 }, boundaryGap: false },
    yAxis: [
      { type: 'value', name: 'Token', axisLabel: { fontSize: 11, formatter: v => v >= 1000 ? (v / 1000).toFixed(0) + 'k' : v } },
      { type: 'value', name: '费用', axisLabel: { fontSize: 11, formatter: v => '¥' + (v >= 1 ? v.toFixed(0) : fix4(v)) }, splitLine: { show: false } }
    ],
    series: [
      { name: '总Token', type: 'line', smooth: true, yAxisIndex: 0, data: rows.map(r => r.total_tokens), lineStyle: { width: 2 }, itemStyle: { color: '#333' }, z: 10 },
      { name: '输入', type: 'line', smooth: true, yAxisIndex: 0, data: rows.map(r => r.input_tokens), lineStyle: { width: 1.5 }, itemStyle: { color: '#4a9eff' }, areaStyle: { color: gradientArea(74, 158, 255) } },
      { name: '输出', type: 'line', smooth: true, yAxisIndex: 0, data: rows.map(r => r.output_tokens), lineStyle: { width: 1.5 }, itemStyle: { color: '#36cfc9' }, areaStyle: { color: gradientArea(54, 207, 201) } },
      { name: '缓存命中', type: 'line', smooth: true, yAxisIndex: 0, data: rows.map(r => r.cached_tokens), lineStyle: { width: 1.5 }, itemStyle: { color: '#95de64' }, areaStyle: { color: gradientArea(149, 222, 100) } },
      { name: '推理', type: 'line', smooth: true, yAxisIndex: 0, data: rows.map(r => r.reasoning_tokens), lineStyle: { width: 1.5 }, itemStyle: { color: '#b37feb' }, areaStyle: { color: gradientArea(179, 127, 235) } },
      { name: '费用', type: 'line', smooth: true, symbol: 'none', yAxisIndex: 1, data: rows.map(r => r.cost), lineStyle: { width: 2, type: 'dashed' }, itemStyle: { color: '#e0a040' } }
    ]
  }
}

// ===== 维度环形图 =====

/**
 * 构建维度分析环形图 option
 * @param {Array} rows — 后端返回的统计行
 * @param {string} metric — 'cost' | 'total_tokens' | 'cached_tokens'
 * @param {object} summary — 顶部汇总数据
 */
export function buildDimensionOption(rows, metric, summary) {
  const mc = metricConfig[metric]
  const sorted = [...rows].sort((a, b) => (b[metric] || 0) - (a[metric] || 0))
  const total = sorted.reduce((s, r) => s + (Number(r[metric]) || 0), 0)

  // 比率指标用水平柱状图
  if (mc.isRate) {
    const labels = sorted.map(r => r.sub_label || r.label)
    return {
      tooltip: { trigger: 'axis', formatter: ps => ps[0].axisValue + '<br/>' + ps[0].marker + mc.label + ': ' + mc.format(ps[0].value) },
      grid: { top: 20, left: 140, right: 16, bottom: 28 },
      xAxis: { type: 'value', min: 0, max: 1, axisLabel: { fontSize: 11, formatter: v => (v * 100).toFixed(0) + '%' } },
      yAxis: { type: 'category', data: labels, axisLabel: { fontSize: 11, width: 120, overflow: 'truncate' } },
      series: [
        { name: mc.label, type: 'bar', data: sorted.map(r => Number(r[metric]) || 0), itemStyle: { color: '#73d13d', borderRadius: [0, 4, 4, 0] }, label: { show: true, position: 'right', formatter: p => mc.format(p.value), fontSize: 11 } }
      ]
    }
  }

  // 非比率指标用环形图
  return {
    tooltip: { trigger: 'item', formatter: pieFormatter(mc) },
    legend: { top: 0, type: 'scroll' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      center: ['50%', '56%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { formatter: p => {
          const n = p.name.length > 10 ? p.name.slice(0, 10) + '…' : p.name
          return n + '\n' + mc.prefix + mc.format(p.value)
        }, fontSize: 11 },
      emphasis: { label: { fontSize: 13, fontWeight: 'bold' } },
      data: sorted.map(r => ({
        name: r.sub_label || r.label,
        value: Number(r[metric]) || 0
      }))
    }],
    graphic: [{
      type: 'text',
      left: 'center',
      top: '44%',
      style: { text: mc.prefix + mc.format(total), fontSize: 16, fontWeight: 600, fill: '#333', textAlign: 'center' }
    }, {
      type: 'text',
      left: 'center',
      top: '52%',
      style: { text: '总' + mc.label, fontSize: 11, fill: '#999', textAlign: 'center' }
    }]
  }
}
