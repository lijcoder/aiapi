import { onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'

/**
 * useChart — ECharts 生命周期管理
 * @param {Ref<HTMLElement|null>} elRef — 图表容器 DOM ref
 * @param {Ref|Array<Ref>} watchSource — 数据变化时自动重绘
 * @param {(chart: echarts.ECharts) => void} renderFn — 渲染逻辑，调用方在此 setOption
 * @returns {{ chart: Ref<echarts.ECharts|null> }}
 */
export function useChart(elRef, watchSource, renderFn) {
  let chart = null

  function getChart() {
    if (!elRef.value) return null
    if (!chart) chart = echarts.init(elRef.value)
    return chart
  }

  function onResize() { chart && chart.resize() }

  onMounted(() => {
    window.addEventListener('resize', onResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', onResize)
    if (chart) { chart.dispose(); chart = null }
  })

  watch(watchSource, () => {
    nextTick(() => {
      const c = getChart()
      if (!c) return
      if (!hasData(watchSource)) { c.clear(); return }
      renderFn(c)
    })
  }, { immediate: true })

  return { chart }
}

function hasData(source) {
  if (Array.isArray(source)) {
    return source.some(s => {
      const v = s.value !== undefined ? s.value : s
      return Array.isArray(v) ? v.length > 0 : !!v
    })
  }
  const v = source.value !== undefined ? source.value : source
  return Array.isArray(v) ? v.length > 0 : !!v
}
