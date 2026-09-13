<template>
  <div class="condition-view">
    <div v-if="root" class="condition-title-row">
      <span class="condition-title">适用条件</span>
      <span v-if="condition?.op" class="match-mode">（{{ modeText }}）</span>
    </div>

    <div v-if="condition?.op" class="condition-list" :class="{ 'nested-list': !root }">
      <template v-for="(child, index) in condition.children || []" :key="index">
        <div v-if="child?.op" class="condition-group-entry">
          <div class="nested-group-title"><span class="match-mode">{{ child.op.toLowerCase() === 'or' ? '满足任一条件' : '全部满足' }}</span></div>
          <PricingConditionView :condition="child" :root="false" />
        </div>
        <PricingConditionView v-else :condition="child" :root="false" />
      </template>
    </div>

    <div v-else-if="condition" class="condition-leaf-view">
      <span class="condition-kind">{{ leaf.label }}：</span>
      <span class="condition-value">{{ leaf.value }}</span>
    </div>
    <div v-else-if="root" class="condition-empty">未设置有效条件</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

defineOptions({ name: 'PricingConditionView' })
const props = defineProps({ condition: { type: Object, default: null }, root: { type: Boolean, default: true } })
const weekdayNames = ['', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六', '星期日']
const tokenNames = { gt: '大于', gte: '大于等于', lt: '小于', lte: '小于等于' }
const modeText = computed(() => props.condition?.op?.toLowerCase() === 'or' ? '满足任一条件' : '全部满足')
const leaf = computed(() => {
  const c = props.condition || {}
  if (c.type === 'weekday') return { label: '星期', value: (c.values || []).map(v => weekdayNames[v] || v).join('、') || '未设置' }
  if (c.type === 'time_range') return { label: '时间段', value: `${c.start || '--:--'} 至 ${c.end || '--:--'}` }
  if (c.type === 'month_day') return { label: '日期', value: (c.month_days || []).map(v => v.replace('-', ' 月 ') + ' 日').join('、') || '未设置' }
  if (c.type === 'total_tokens') return { label: '总 Token', value: `${tokenNames[c.operator] || c.operator || ''} ${Number(c.value || 0).toLocaleString()} Token` }
  return { label: '条件', value: '未知条件' }
})
</script>

<style scoped>
.condition-view { width: 100%; }
.condition-title-row { display: flex; align-items: center; gap: 5px; margin-bottom: 9px; }
.condition-title { color: #303133; font-size: 13px; font-weight: 700; }
.match-mode { color: #d03050; font-size: 13px; font-weight: 600; }
.condition-list { display: flex; flex-direction: column; gap: 8px; }
.nested-list { margin: 7px 0 0 10px; padding-left: 10px; border-left: 2px solid #e5e7eb; }
.condition-group-entry { padding: 8px 10px; border: 1px solid #e5e7eb; border-radius: 7px; background: #fafafa; }
.nested-group-title { margin-bottom: 3px; }
.condition-leaf-view { display: flex; align-items: baseline; gap: 2px; min-height: 0; padding: 4px 9px; border: 1px solid #ebeef5; border-radius: 6px; background: #fff; font-size: 13px; line-height: 1.35; }
.condition-kind { flex: 0 0 auto; color: #606266; font-weight: 600; }
.condition-value { color: #303133; word-break: break-word; }
.condition-empty { color: #909399; font-size: 13px; }
</style>
