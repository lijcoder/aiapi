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
        table-layout="auto"
        :scroll-x="1310"
        :pagination="pagination"
        :remote="true"
        @update:page="onPage"
        @update:page-size="onPageSize"
        style="width:100%"
      />
    </n-card>

    <!-- 新增/复制/编辑弹窗 -->
    <n-modal v-model:show="showForm" preset="card" :title="formType==='create'?'新增模型':formType==='copy'?'复制模型':'编辑模型'" style="width:860px" :mask-closable="false">
      <div class="model-form">
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
          <n-space>
            <n-checkbox :checked="modalFlags.includes('text')" @update:checked="value => toggleModalFlag('text', value)" label="文本" />
            <n-checkbox :checked="modalFlags.includes('image')" @update:checked="value => toggleModalFlag('image', value)" label="图像" />
            <n-checkbox :checked="modalFlags.includes('video')" @update:checked="value => toggleModalFlag('video', value)" label="视频" />
          </n-space>
        </div>

        <n-divider style="margin:2px 0 0" />

        <!-- 计费配置放在模型基础配置之后，使用图形化表单生成 pricing_config JSON -->
        <div class="pricing-heading">
          <div>
            <div style="font-size:15px;font-weight:600">分段计费</div>
            <div class="form-help">价格单位：元/百万 Token。规则中启用的条件需同时满足，多个命中时按优先级最高的规则计费。</div>
          </div>
          <n-button size="small" type="primary" secondary @click="addRule">新增规则</n-button>
        </div>

        <div class="price-row">
          <div class="price-title">默认价格（未命中规则时使用）</div>
          <div class="price-fields">
            <div><div class="field-label">输入（缓存命中）</div><n-input-number v-model:value="form.pricing.default_price.input_cache_hit" :min="0" :precision="6" style="width:100%" /></div>
            <div><div class="field-label">输入（缓存未命中）</div><n-input-number v-model:value="form.pricing.default_price.input_cache_miss" :min="0" :precision="6" style="width:100%" /></div>
            <div><div class="field-label">输出</div><n-input-number v-model:value="form.pricing.default_price.output" :min="0" :precision="6" style="width:100%" /></div>
          </div>
        </div>

        <div class="timezone-row">
          <div class="field-label">计费时区</div>
          <n-input v-model:value="form.pricing.timezone" placeholder="Asia/Shanghai" style="width:240px" />
          <span class="form-help">按请求开始时间匹配；留空时使用 Asia/Shanghai</span>
        </div>

        <n-empty v-if="!form.pricing.rules.length" description="暂无分段规则，当前只使用默认价格" size="small" />
        <n-card v-for="(rule, index) in form.pricing.rules" :key="rule._key" size="small" class="rule-card" :bordered="true">
          <template #header>
            <n-space align="center"><span>规则 {{ index + 1 }}</span><n-tag v-if="rule.name" size="small" :bordered="false" type="info">{{ rule.name }}</n-tag></n-space>
          </template>
          <template #header-extra>
            <n-space align="center" size="small">
              <n-checkbox v-model:checked="rule.enabled">启用</n-checkbox>
              <n-button size="small" tertiary type="error" @click="removeRule(index)">删除</n-button>
            </n-space>
          </template>

          <div class="rule-grid">
            <div><div class="field-label">规则名称（不可重复）</div><n-input v-model:value="rule.name" placeholder="如工作日高峰" /></div>
            <div><div class="field-label">优先级</div><n-input-number v-model:value="rule.priority" :precision="0" style="width:100%" /></div>
          </div>

          <div class="rule-section-title">规则价格</div>
          <div class="price-fields">
            <div><div class="field-label">输入（缓存命中）</div><n-input-number v-model:value="rule.price.input_cache_hit" :min="0" :precision="6" style="width:100%" /></div>
            <div><div class="field-label">输入（缓存未命中）</div><n-input-number v-model:value="rule.price.input_cache_miss" :min="0" :precision="6" style="width:100%" /></div>
            <div><div class="field-label">输出</div><n-input-number v-model:value="rule.price.output" :min="0" :precision="6" style="width:100%" /></div>
          </div>

          <div class="rule-section-title">匹配条件（支持嵌套 AND / OR）</div>
          <PricingConditionEditor v-model="rule.when" :removable="false" />
        </n-card>
      </div>
      <p v-if="formMsg" style="color:#d03050;font-size:13px;margin-top:8px">{{ formMsg }}</p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showForm=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doSubmit">确认</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showPricing" preset="card" title="模型计费" style="width:760px">
      <div v-if="pricingDetail" class="pricing-detail">
        <div class="detail-title">{{ pricingDetail.provider }} / {{ pricingDetail.model }}</div>
        <div class="detail-meta">计费时区：{{ pricingDetail.config?.timezone || 'Asia/Shanghai' }}　·　价格单位：元/百万 Token</div>
        <div class="detail-section-title">默认价格</div>
        <div class="detail-price-grid">
          <div><span>输入（缓存命中）</span><strong>¥{{ priceText(pricingDetail.config?.default_price?.input_cache_hit) }}</strong></div>
          <div><span>输入（缓存未命中）</span><strong>¥{{ priceText(pricingDetail.config?.default_price?.input_cache_miss) }}</strong></div>
          <div><span>输出</span><strong>¥{{ priceText(pricingDetail.config?.default_price?.output) }}</strong></div>
        </div>
        <div class="detail-section-title">分段规则（优先级越高越先匹配）</div>
        <n-empty v-if="!pricingDetail.rules.length" description="暂无分段规则，所有请求使用默认价格" size="small" />
        <n-card v-for="(rule, index) in pricingDetail.rules" :key="pricingRuleKey(rule, index)" size="small" class="detail-rule" :bordered="true">
          <template #header>
            <div class="detail-rule-header" @click="toggleRule(rule, index)"><n-space align="center" size="small">
              <span>{{ rule.name || '未命名规则' }}</span>
              <n-tag size="small" :bordered="false" type="info">优先级 {{ rule.priority }}</n-tag>
              <n-tag size="small" :bordered="false" :type="rule.enabled === false ? 'warning' : 'success'">{{ rule.enabled === false ? '已停用' : '已启用' }}</n-tag>
            </n-space></div>
          </template>
          <template #header-extra><n-button size="small" quaternary @click.stop="toggleRule(rule, index)">{{ isRuleExpanded(rule, index) ? '收起' : '展开' }}</n-button></template>
          <div class="detail-rule-price">
            <span><em>输入（缓存命中）</em><strong>¥{{ priceText(rule.price?.input_cache_hit) }}</strong></span>
            <span><em>输入（缓存未命中）</em><strong>¥{{ priceText(rule.price?.input_cache_miss) }}</strong></span>
            <span><em>输出</em><strong>¥{{ priceText(rule.price?.output) }}</strong></span>
          </div>
          <div v-if="isRuleExpanded(rule, index)" class="detail-condition"><PricingConditionView :condition="normalizeCondition(rule.when)" /></div>
        </n-card>
      </div>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NModal, NInput, NInputNumber, NButton, NSpace, NCheckbox, NTag, NDropdown, NDivider, NEmpty, NIcon, useMessage, useDialog } from 'naive-ui'
import { ArrowDownOutline, ArrowUpOutline, FlashOutline } from '@vicons/ionicons5'
import { listModelsAdmin, createModel, updateModel, deleteModel } from '../../api'
import { usePagination } from '../../composables/usePagination'
import { formatTime } from '../../utils'
import PricingConditionEditor from '../../components/PricingConditionEditor.vue'
import PricingConditionView from '../../components/PricingConditionView.vue'

const message = useMessage()
const dialog = useDialog()

const models = ref([])
const tableLoading = ref(false)
const providerKw = ref('')
const modelKw = ref('')
const { pagination, onPage, onPageSize, resetAndLoad } = usePagination(load)

// 新增/编辑
const showForm = ref(false)
const formType = ref('create')
const form = ref(emptyForm())
const formMsg = ref('')
const submitting = ref(false)
const modalFlags = ref(['text'])
const showPricing = ref(false)
const pricingDetail = ref(null)
const expandedRules = ref({})

let ruleKey = 0

function emptyForm() {
  return {
    id: 0,
    provider: '',
    model: '',
    max_context_tokens: 0,
    max_completion_tokens: 0,
    pricing: defaultPricingConfig(),
  }
}

function defaultPricingConfig() {
  return {
    version: 2,
    timezone: 'Asia/Shanghai',
    default_price: {
      input_cache_hit: 0,
      input_cache_miss: 0,
      output: 0,
    },
    rules: [],
  }
}

function newRule() {
  ruleKey += 1
  return {
    _key: `rule-${Date.now()}-${ruleKey}`,
    name: '',
    enabled: true,
    priority: 100 - ruleKey,
    price: { input_cache_hit: 0, input_cache_miss: 0, output: 0 },
    when: { op: 'and', children: [] },
  }
}

function pricingToForm(raw) {
  let parsed
  try { parsed = raw ? JSON.parse(raw) : defaultPricingConfig() } catch { parsed = defaultPricingConfig() }
  const pricing = {
    version: 2,
    timezone: parsed.timezone || 'Asia/Shanghai',
    default_price: {
      input_cache_hit: parsed.default_price?.input_cache_hit ?? 0,
      input_cache_miss: parsed.default_price?.input_cache_miss ?? 0,
      output: parsed.default_price?.output ?? 0,
    },
    rules: [],
  }
  for (const source of (parsed.rules || [])) {
    const rule = newRule()
    rule.name = source.name || source.id || ''
    rule.enabled = source.enabled !== false
    rule.priority = source.priority ?? 0
    rule.price = {
      input_cache_hit: source.price?.input_cache_hit ?? 0,
      input_cache_miss: source.price?.input_cache_miss ?? 0,
      output: source.price?.output ?? 0,
    }
    rule.when = source.when?.op || source.when?.type ? source.when : legacyWhen(source.when)
    pricing.rules.push(rule)
  }
  return pricing
}

function pricingFromForm(pricing) {
  return {
    version: 2,
    timezone: pricing.timezone || 'Asia/Shanghai',
    default_price: {
      input_cache_hit: pricing.default_price.input_cache_hit ?? 0,
      input_cache_miss: pricing.default_price.input_cache_miss ?? 0,
      output: pricing.default_price.output ?? 0,
    },
    rules: pricing.rules.map(rule => ({
        name: rule.name,
        enabled: rule.enabled,
        priority: rule.priority ?? 0,
        when: stripKeys(rule.when),
        price: {
          input_cache_hit: rule.price.input_cache_hit ?? 0,
          input_cache_miss: rule.price.input_cache_miss ?? 0,
          output: rule.price.output ?? 0,
        },
      })),
  }
}

function legacyWhen(when) {
  if (!when) return { op: 'and', children: [] }
  const children = []
  if (when.time) {
    if (when.time.weekdays?.length) children.push({ type: 'weekday', values: when.time.weekdays })
    const windows = (when.time.windows || []).map(w => ({ type: 'time_range', start: w.start, end: w.end }))
    children.push(windows.length === 1 ? windows[0] : { op: 'or', children: windows })
  }
  if (when.total_tokens) {
    for (const op of ['gt', 'gte', 'lt', 'lte']) if (when.total_tokens[op] != null) children.push({ type: 'total_tokens', operator: op, value: when.total_tokens[op] })
  }
  return children.length === 1 ? children[0] : { op: 'and', children }
}

function stripKeys(node) {
  if (!node) return node
  const result = { ...node }
  delete result._key
  if (result.children) result.children = result.children.map(stripKeys)
  return result
}

function parsePricingConfig(config) {
  try {
    return typeof config === 'string' ? JSON.parse(config) : config
  } catch {
    return null
  }
}

function renderPricing(row) {
  const parsed = parsePricingConfig(row.pricing_config)
  if (!parsed) return h('span', { class: 'pricing-unconfigured' }, '未配置')
  const price = parsed.default_price || {}
  const rules = Array.isArray(parsed.rules) ? parsed.rules : []
  const rate = (icon, value, label, className) => h('span', {
    class: ['pricing-rate', className],
    title: `${label}：¥${priceText(value)}`,
  }, [
    h(NIcon, { size: 15 }, { default: () => h(icon) }),
    h('span', { class: 'pricing-value' }, `¥${priceText(value)}`),
  ])
  return h('div', {
    class: 'pricing-summary',
    role: 'button',
    tabindex: 0,
    title: '点击查看计费详情',
    'aria-label': '点击查看计费详情',
    onClick: () => openPricing(row),
    onKeydown: (event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        openPricing(row)
      }
    },
  }, [
    h('span', { class: 'pricing-rule-count', title: `${rules.length} 条分段规则` }, String(rules.length)),
    h('span', { class: 'pricing-rate-list' }, [
      rate(FlashOutline, price.input_cache_hit, '输入（缓存命中）', 'pricing-rate-cache'),
      rate(ArrowUpOutline, price.input_cache_miss, '输入（缓存未命中）', 'pricing-rate-input'),
      rate(ArrowDownOutline, price.output, '输出', 'pricing-rate-output'),
    ]),
  ])
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

function toggleModalFlag(flag, checked) {
  const next = new Set(modalFlags.value)
  if (checked) next.add(flag)
  else next.delete(flag)
  modalFlags.value = [...next]
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
  { title: '模型', key: 'model', width: 200, render(r) { return h('span', { style: 'display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap', title: r.model }, r.model) } },
  { title: '计费配置', key: 'pricing_config', width: 340, render: renderPricing },
  { title: '上下文', key: 'max_context_tokens', width: 90, render(r) { return fmtK(r.max_context_tokens) } },
  { title: '最大输出', key: 'max_completion_tokens', width: 90, render(r) { return fmtK(r.max_completion_tokens) } },
  { title: '能力', key: 'modal', width: 140, render: renderModal },
  { title: '创建时间', key: 'created_at', width: 170, render(r) { const value = formatTime(r.created_at); return h('span', { style: 'display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap', title: value }, value) } },
  { title: '操作', key: 'actions', minWidth: 150, fixed: 'right', render(r) {
    const moreOptions = [
      { label: '编辑', key: 'edit' },
      { label: '复制', key: 'copy' },
      { label: '删除', key: 'delete' },
    ]
    function onSelect(key) {
      if (key === 'edit') openEdit(r)
      else if (key === 'copy') openCopy(r)
      else if (key === 'delete') onDelete(r)
    }
    return h('div', { class: 'model-actions' }, [
      h(NButton, { size: 'small', tertiary: true, type: 'info', onClick: () => openPricing(r) }, () => '查看计费'),
      h(NDropdown, { options: moreOptions, trigger: 'click', onSelect: (k) => onSelect(k) }, {
        default: () => h(NButton, { size: 'small', tertiary: true }, () => '更多')
      }),
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
    max_context_tokens: r.max_context_tokens,
    max_completion_tokens: r.max_completion_tokens,
    pricing: pricingToForm(r.pricing_config),
  }
  modalFlags.value = modalToFlags(r)
  formMsg.value = ''
  showForm.value = true
}

// 复制：以某行为模板打开新增弹窗，用户可修改（provider/model 可编辑）后提交新增
function openCopy(r) {
  formType.value = 'copy'
  form.value = {
    id: 0,
    provider: r.provider,
    model: r.model,
    max_context_tokens: r.max_context_tokens,
    max_completion_tokens: r.max_completion_tokens,
    pricing: pricingToForm(r.pricing_config),
  }
  modalFlags.value = modalToFlags(r)
  formMsg.value = ''
  showForm.value = true
}

function addRule() {
  form.value.pricing.rules.push(newRule())
}

function removeRule(index) {
  form.value.pricing.rules.splice(index, 1)
}

function priceText(value) {
  const number = Number(value)
  return Number.isFinite(number) ? number.toFixed(2) : '0.00'
}

function normalizeCondition(when) {
  if (!when) return null
  if (when.op || when.type) return when
  const children = []
  if (when.time) {
    if (when.time.weekdays?.length) children.push({ type: 'weekday', values: when.time.weekdays })
    const windows = (when.time.windows || []).map(w => ({ type: 'time_range', start: w.start, end: w.end }))
    if (windows.length === 1) children.push(windows[0]); else if (windows.length > 1) children.push({ op: 'or', children: windows })
  }
  if (when.total_tokens) for (const op of ['gt', 'gte', 'lt', 'lte']) if (when.total_tokens[op] != null) children.push({ type: 'total_tokens', operator: op, value: when.total_tokens[op] })
  return children.length === 1 ? children[0] : { op: 'and', children }
}

function openPricing(row) {
  let config = null
  try { config = JSON.parse(row.pricing_config) } catch {}
  const rules = config?.rules ? [...config.rules].sort((a, b) => (b.priority || 0) - (a.priority || 0)) : []
  pricingDetail.value = {
    provider: row.provider,
    model: row.model,
    config,
    rules,
  }
  expandedRules.value = Object.fromEntries(rules.map((rule, index) => [pricingRuleKey(rule, index), rules.length === 1]))
  showPricing.value = true
}

function pricingRuleKey(rule, index) { return `${rule.name || 'rule'}-${index}` }
function isRuleExpanded(rule, index) { return expandedRules.value[pricingRuleKey(rule, index)] === true }
function toggleRule(rule, index) {
  const key = pricingRuleKey(rule, index)
  expandedRules.value[key] = !isRuleExpanded(rule, index)
}

function validatePricingForm() {
  const pricing = form.value.pricing
  if (!pricing.timezone.trim()) pricing.timezone = 'Asia/Shanghai'
  const priorities = new Set()
  for (let index = 0; index < pricing.rules.length; index += 1) {
    const rule = pricing.rules[index]
    const label = `规则 ${index + 1}`
    if (!rule.name.trim()) return `${label} 的规则名称不能为空`
    if (pricing.rules.some((item, i) => i !== index && item.name.trim() === rule.name.trim())) return `${label} 的规则名称重复`
    if (priorities.has(rule.priority)) return `${label} 的优先级重复，请调整后再保存`
    priorities.add(rule.priority)
    const error = validateCondition(rule.when)
    if (error) return `${label}：${error}`
  }
  return ''
}

function validateCondition(node) {
  if (!node) return '至少添加一个匹配条件'
  if (node.op) {
    if (!['and', 'or'].includes(node.op)) return '条件组必须是 AND 或 OR'
    if (!node.children?.length) return '条件组不能为空'
    for (const child of node.children) { const error = validateCondition(child); if (error) return error }
    return ''
  }
  if (node.type === 'weekday' && !node.values?.length) return '星期条件至少选择一天'
  if (node.type === 'time_range') {
    if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(node.start || '') || !/^([01]\d|2[0-3]):[0-5]\d$/.test(node.end || '')) return '时间段必须使用 HH:MM 格式'
    if (node.start >= node.end) return '时间段必须满足开始时间早于结束时间'
  }
  if (node.type === 'month_day' && !node.month_days?.length) return '日期条件至少选择一天'
  if (node.type === 'total_tokens' && (!['gt', 'gte', 'lt', 'lte'].includes(node.operator) || node.value == null || node.value < 0)) return 'Token 条件无效'
  return ''
}

async function doSubmit() {
  formMsg.value = ''
  if (formType.value !== 'edit') {
    if (!form.value.provider || !form.value.model) {
      formMsg.value = 'provider 和 model 必填'
      return
    }
  }
  const pricingError = validatePricingForm()
  if (pricingError) {
    formMsg.value = pricingError
    return
  }
  submitting.value = true
  try {
    if (formType.value !== 'edit') {
      await createModel({
        provider: form.value.provider,
        model: form.value.model,
        pricing_config: JSON.stringify(pricingFromForm(form.value.pricing)),
        max_context_tokens: form.value.max_context_tokens || 0,
        max_completion_tokens: form.value.max_completion_tokens || 0,
        ...flagsToModal(modalFlags.value),
      })
      message.success('创建成功')
    } else {
      await updateModel({
        id: form.value.id,
        pricing_config: JSON.stringify(pricingFromForm(form.value.pricing)),
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

<style scoped>
.model-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 4px;
}

.pricing-heading,
.timezone-row,
.window-row,
.token-condition-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pricing-heading {
  justify-content: space-between;
}

.price-row {
  border: 1px solid var(--n-border-color);
  border-radius: 3px;
  padding: 12px;
}

.price-title,
.rule-section-title,
.condition-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 10px;
}

.price-fields {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.rule-card :deep(.n-card__content) {
  padding-top: 4px;
}

.rule-grid {
  display: grid;
  grid-template-columns: 1fr 1.5fr 120px;
  gap: 10px;
}

.rule-section-title {
  margin-top: 16px;
}

.condition-box {
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  padding: 12px;
  margin-top: 10px;
}

.condition-box:first-of-type {
  margin-top: 0;
}

.field-label {
  color: #606266;
  font-size: 12px;
  margin-bottom: 5px;
}

.form-help {
  color: #909399;
  font-size: 12px;
}

.model-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.window-label {
  margin-top: 12px;
}

.window-row {
  margin-bottom: 8px;
}

.condition-and {
  color: #909399;
  flex: 0 0 auto;
}

.pricing-detail {
  max-height: 65vh;
  overflow-y: auto;
  padding-right: 4px;
}

:global(.pricing-summary) {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-width: 0;
  padding: 2px 3px;
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
  transition: background-color .15s ease;
}

:global(.pricing-summary:hover),
:global(.pricing-summary:focus-visible) {
  outline: none;
  background: var(--n-hover-color);
}

:global(.pricing-rule-count) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  min-width: 28px;
  height: 28px;
  padding: 0 5px;
  border-radius: 3px;
  background: #d03050;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 28px;
  box-shadow: 0 1px 2px rgba(208, 48, 80, .2);
}

:global(.pricing-rate-list) {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

:global(.pricing-rate) {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex: 0 0 86px;
  box-sizing: border-box;
  min-height: 28px;
  padding: 2px 7px 2px 6px;
  border: 1px solid transparent;
  border-radius: 3px;
  white-space: nowrap;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 18px;
  transition: filter .15s ease;
}

:global(.pricing-rate:hover) { filter: brightness(.97); }
:global(.pricing-rate-cache) { color: #16834a; background: rgba(24, 160, 88, .09); border-color: rgba(24, 160, 88, .2); }
:global(.pricing-rate-input) { color: #1769aa; background: rgba(32, 128, 240, .09); border-color: rgba(32, 128, 240, .2); }
:global(.pricing-rate-output) { color: #7043c3; background: rgba(138, 92, 246, .09); border-color: rgba(138, 92, 246, .2); }
:global(.pricing-rate .n-icon) { flex: 0 0 auto; }
:global(.pricing-value) { flex: 1; color: currentColor; font-weight: 600; text-align: left; }
:global(.pricing-unconfigured) { color: var(--n-text-color-3); }

.detail-title {
  font-size: 16px;
  font-weight: 600;
}

.detail-meta,
.detail-price-grid span {
  color: #909399;
  font-size: 12px;
}

.detail-rule-price { display: flex; flex-wrap: wrap; gap: 8px 18px; padding: 9px 10px; border-radius: 6px; background: #f7f8fa; color: #606266; font-size: 13px; line-height: 1.6; }
.detail-rule-price span { display: inline-flex; align-items: baseline; gap: 6px; }
.detail-rule-price em { color: #909399; font-size: 12px; font-style: normal; }
.detail-rule-price strong { color: #303133; font-size: 14px; }

.detail-condition-title {
  margin: 14px 0 7px;
  color: #606266;
  font-size: 12px;
  font-weight: 600;
}

.detail-meta {
  margin-top: 6px;
}

.detail-section-title {
  font-size: 13px;
  font-weight: 600;
  margin: 18px 0 10px;
}

.detail-price-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.detail-price-grid > div {
  display: flex;
  flex-direction: column;
  gap: 5px;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  padding: 10px;
}

.detail-price-grid strong {
  font-size: 15px;
}

.detail-rule {
  margin-bottom: 10px;
}

.detail-rule-header {
  cursor: pointer;
}

.detail-condition {
  line-height: 1.35;
  margin-bottom: 6px;
}

@media (max-width: 720px) {
  .price-fields,
  .rule-grid {
    grid-template-columns: 1fr;
  }

  .token-condition-row {
    flex-wrap: wrap;
  }

  .detail-price-grid {
    grid-template-columns: 1fr;
  }
}
</style>
