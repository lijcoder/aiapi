<template>
  <n-card title="模型列表" size="small">
    <template #header-extra>
      <n-space align="center">
        <n-input v-model:value="providerKw" placeholder="提供商" size="small" clearable style="width:140px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
        <n-input v-model:value="modelKw" placeholder="模型" size="small" clearable style="width:160px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
        <n-button size="small" @click="resetAndLoad">查询</n-button>
      </n-space>
    </template>
    <n-data-table :columns="columns" :data="models" :loading="loading" :bordered="false" size="small" :scroll-x="1120" :pagination="pagination" :remote="true" @update:page="onPage" @update:page-size="onPageSize" style="width:100%" />
  </n-card>

  <n-modal v-model:show="showPricing" preset="card" title="模型计费" style="width:760px">
    <div v-if="pricingDetail" class="pricing-detail">
      <div class="detail-title">{{ pricingDetail.provider }} / {{ pricingDetail.model }}</div>
      <template v-if="pricingDetail.config">
        <div class="detail-meta">计费时区：{{ pricingDetail.config.timezone || 'Asia/Shanghai' }}　·　价格单位：元/百万 Token</div>
        <div class="detail-section-title">默认价格</div>
        <div class="detail-price-grid">
          <div><span>输入（缓存命中）</span><strong>¥{{ priceText(pricingDetail.config.default_price?.input_cache_hit) }}</strong></div>
          <div><span>输入（缓存未命中）</span><strong>¥{{ priceText(pricingDetail.config.default_price?.input_cache_miss) }}</strong></div>
          <div><span>输出</span><strong>¥{{ priceText(pricingDetail.config.default_price?.output) }}</strong></div>
        </div>

        <div class="detail-section-title">分段规则（优先级越高越先匹配）</div>
        <n-empty v-if="!pricingDetail.rules.length" description="暂无分段规则，所有请求使用默认价格" size="small" />
        <n-card v-for="(rule, index) in pricingDetail.rules" :key="ruleKey(rule, index)" size="small" class="detail-rule" :bordered="true">
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
      </template>
      <n-empty v-else description="该模型尚未配置有效的计费规则" size="small" />
    </div>
  </n-modal>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NInput, NButton, NSpace, NTag, NModal, NEmpty } from 'naive-ui'
import { listModels } from '../api'
import { usePagination } from '../composables/usePagination'
import { formatTime } from '../utils'
import PricingConditionView from '../components/PricingConditionView.vue'

const columns = [
  { title: '提供商', key: 'provider', width: 110 },
  { title: '模型', key: 'model', width: 200, ellipsis: { tooltip: true } },
  { title: '计费配置', key: 'pricing_config', width: 260, ellipsis: { tooltip: true }, render(r) { return pricingSummary(r.pricing_config) }},
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
  { title: '操作', key: 'actions', width: 110, fixed: 'right', render(r) {
    return h(NButton, { size: 'small', tertiary: true, type: 'info', onClick: () => openPricing(r) }, () => '查看计费')
  }},
]

function pricingSummary(config) {
  try {
    const parsed = JSON.parse(config)
    const price = parsed.default_price || {}
    return `默认 ¥${price.input_cache_hit ?? 0} / ¥${price.input_cache_miss ?? 0} / ¥${price.output ?? 0}；${(parsed.rules || []).length} 条规则`
  } catch {
    return '未配置'
  }
}

const showPricing = ref(false)
const pricingDetail = ref(null)
const expandedRules = ref({})

function priceText(value) {
  if (value == null) return '0'
  return Number(value).toFixed(6).replace(/\.?0+$/, '')
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
  expandedRules.value = Object.fromEntries(rules.map((rule, index) => [ruleKey(rule, index), rules.length === 1]))
  showPricing.value = true
}

function ruleKey(rule, index) { return `${rule.name || 'rule'}-${index}` }
function isRuleExpanded(rule, index) { return expandedRules.value[ruleKey(rule, index)] === true }
function toggleRule(rule, index) {
  const key = ruleKey(rule, index)
  expandedRules.value[key] = !isRuleExpanded(rule, index)
}

const models = ref([])
const loading = ref(false)
const providerKw = ref('')
const modelKw = ref('')
const { pagination, onPage, onPageSize, resetAndLoad } = usePagination(load)

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

<style scoped>
.pricing-detail {
  max-height: 65vh;
  overflow-y: auto;
  padding-right: 4px;
}

.detail-title {
  font-size: 16px;
  font-weight: 600;
}

.detail-meta {
  color: #909399;
  font-size: 12px;
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

@media (max-width: 640px) {
  .detail-price-grid {
    grid-template-columns: 1fr;
  }
}
</style>
