<template>
  <div class="condition-editor">
    <div v-if="isGroup" class="condition-group">
      <div class="group-toolbar">
        <span class="node-label">条件组</span>
        <n-select v-model:value="node.op" :options="groupOptions" size="small" style="width:110px" />
        <n-button size="small" dashed @click="addLeaf">添加条件</n-button>
        <n-button size="small" dashed @click="addGroup">添加条件组</n-button>
        <n-button v-if="removable" size="small" quaternary type="error" @click="$emit('remove')">删除</n-button>
      </div>
      <div class="children">
        <div v-for="(child, index) in node.children" :key="child._key || index" class="child-row">
          <PricingConditionEditor v-model="node.children[index]" :removable="true" @remove="removeChild(index)" />
        </div>
        <div v-if="!node.children.length" class="empty-hint">请添加至少一个条件</div>
      </div>
    </div>

    <div v-else class="condition-leaf">
      <div class="leaf-toolbar">
        <span class="node-label">条件</span>
        <n-select :value="node.type" :options="leafOptions" size="small" style="width:150px" @update:value="changeType" />
        <n-button v-if="removable" size="small" quaternary type="error" @click="$emit('remove')">删除</n-button>
      </div>

      <div v-if="node.type === 'weekday'" class="leaf-fields">
        <span class="field-label">选择适用星期（可多选）</span>
        <n-checkbox-group v-model:value="node.values"><n-space wrap><n-checkbox v-for="item in weekdays" :key="item.value" :value="item.value" :label="item.label" /></n-space></n-checkbox-group>
      </div>
      <div v-else-if="node.type === 'time_range'" class="leaf-fields">
        <span class="field-label">每日生效时段</span>
        <div class="time-range-control">
          <label class="time-field"><span>开始</span><n-time-picker :value="clockToValue(node.start)" format="HH:mm" :actions="null" clearable placeholder="09:00" @update:value="v => node.start = valueToClock(v)" /></label>
          <span class="range-arrow">→</span>
          <label class="time-field"><span>结束</span><n-time-picker :value="clockToValue(node.end)" format="HH:mm" :actions="null" clearable placeholder="18:00" @update:value="v => node.end = valueToClock(v)" /></label>
        </div>
      </div>
      <div v-else-if="node.type === 'month_day'" class="leaf-fields">
        <span class="field-label">选择日期</span>
        <div class="date-control">
          <n-date-picker :value="dateToAdd" type="date" :actions="null" clearable placeholder="从日历选择日期" @update:value="addDate" />
          <span class="date-help">每年同一日期均生效</span>
        </div>
        <n-space v-if="node.month_days?.length" class="date-tags" size="small" wrap>
          <n-tag v-for="date in node.month_days" :key="date" closable @close="removeDate(date)">{{ displayDate(date) }}</n-tag>
        </n-space>
      </div>
      <div v-else class="leaf-fields">
        <span class="field-label">按本次请求总 Token 判断</span>
        <div class="token-control">
          <span class="token-subject">总 Token</span>
          <n-radio-group v-model:value="node.operator" size="small">
            <n-radio-button value="gt">&gt; 大于</n-radio-button>
            <n-radio-button value="gte">≥ 大于等于</n-radio-button>
            <n-radio-button value="lt">&lt; 小于</n-radio-button>
            <n-radio-button value="lte">≤ 小于等于</n-radio-button>
          </n-radio-group>
          <n-input-number v-model:value="node.value" :min="0" :precision="0" placeholder="输入数量" style="width:180px" />
          <span class="token-unit">Token</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { NButton, NCheckbox, NCheckboxGroup, NDatePicker, NInputNumber, NRadioButton, NRadioGroup, NSelect, NSpace, NTag, NTimePicker } from 'naive-ui'

defineOptions({ name: 'PricingConditionEditor' })
const props = defineProps({ modelValue: { type: Object, required: true }, removable: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue', 'remove'])
const node = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
const isGroup = computed(() => !!node.value.op)

const groupOptions = [{ label: '全部满足（AND）', value: 'and' }, { label: '任一满足（OR）', value: 'or' }]
const leafOptions = [
  { label: '星期', value: 'weekday' },
  { label: '时间段', value: 'time_range' },
  { label: '日期', value: 'month_day' },
  { label: '请求 Token', value: 'total_tokens' },
]
const weekdays = [{ value: 1, label: '周一' }, { value: 2, label: '周二' }, { value: 3, label: '周三' }, { value: 4, label: '周四' }, { value: 5, label: '周五' }, { value: 6, label: '周六' }, { value: 7, label: '周日' }]
const dateToAdd = ref(null)

let keySeed = 0
function mark(node) { if (!node._key) { keySeed += 1; node._key = `condition-${Date.now()}-${keySeed}` } return node }
function addLeaf() { node.value.children.push(mark({ type: 'weekday', values: [] })) }
function addGroup() { node.value.children.push(mark({ op: 'and', children: [] })) }
function removeChild(index) { node.value.children.splice(index, 1) }
function addDate(value) {
  if (value == null) return
  const date = new Date(value)
  const monthDay = `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  if (!node.value.month_days) node.value.month_days = []
  if (!node.value.month_days.includes(monthDay)) node.value.month_days.push(monthDay)
  node.value.month_days.sort()
  dateToAdd.value = null
}
function removeDate(date) { node.value.month_days = node.value.month_days.filter(item => item !== date) }
function displayDate(value) { return value.replace('-', ' 月 ') + ' 日' }
function changeType(type) {
  const next = { type }
  if (type === 'weekday') next.values = []
  if (type === 'time_range') { next.start = ''; next.end = '' }
  if (type === 'month_day') next.month_days = []
  if (type === 'total_tokens') { next.operator = 'gte'; next.value = 0 }
  Object.keys(node.value).forEach(key => { if (key !== '_key') delete node.value[key] })
  Object.assign(node.value, next)
}
function clockToValue(clock) {
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(clock || '')
  if (!match) return null
  const date = new Date()
  date.setHours(Number(match[1]), Number(match[2]), 0, 0)
  return date.getTime()
}
function valueToClock(value) {
  if (value == null) return ''
  const date = new Date(value)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
</script>

<style scoped>
.condition-group, .condition-leaf { border: 1px solid var(--n-border-color); border-radius: 6px; padding: 10px; background: var(--n-color); }
.group-toolbar, .leaf-toolbar { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.node-label { font-size:12px; font-weight:600; color:#606266; }
.children { margin:10px 0 0 12px; padding-left:12px; border-left:2px solid var(--n-border-color); display:flex; flex-direction:column; gap:8px; }
.empty-hint { color:#909399; font-size:12px; }
.leaf-fields { margin-top:10px; display:flex; flex-direction:column; gap:6px; }
.field-label { color:#909399; font-size:12px; }
.time-range-control, .date-control, .token-control { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.time-field { display:flex; align-items:center; gap:6px; color:#606266; font-size:12px; }
.time-field :deep(.n-time-picker) { width:150px; }
.range-arrow { color:#909399; font-size:18px; }
.date-control :deep(.n-date-picker) { width:220px; }
.date-help, .token-unit { color:#909399; font-size:12px; }
.date-tags { margin-top:2px; }
.token-subject { color:#606266; white-space:nowrap; }
</style>
