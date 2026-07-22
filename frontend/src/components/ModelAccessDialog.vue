<template>
  <n-modal v-model:show="show" preset="card" :title="title" style="width:680px" :mask-closable="false">
    <div style="display:flex;flex-direction:column;gap:14px">
      <!-- 策略选择 -->
      <n-radio-group v-model:value="form.policy">
        <n-space>
          <n-radio value="all">全量模型（放行所有模型）</n-radio>
          <n-radio value="whitelist">指定模型（白名单）</n-radio>
        </n-space>
      </n-radio-group>

      <!-- 白名单配置区 -->
      <template v-if="form.policy === 'whitelist'">
        <div style="font-size:13px;color:#909399">
          勾选该 Key 允许访问的模型，不勾选任何模型则该 Key 无法访问任何模型
        </div>

        <!-- 搜索 + 统计 + 批量操作 -->
        <div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap">
          <n-input
            v-model:value="keyword"
            placeholder="搜索模型名"
            size="small"
            clearable
            style="width:200px"
          />
          <span style="font-size:13px;color:#666">
            已选 <b style="color:#1677ff">{{ selectedCount }}</b> / {{ totalCount }} 个
          </span>
          <n-space size="small" style="margin-left:auto">
            <n-button size="tiny" quaternary @click="selectAllFiltered">全选(当前筛选)</n-button>
            <n-button size="tiny" quaternary @click="clearAll">清空</n-button>
          </n-space>
        </div>

        <!-- 模型列表 -->
        <div v-if="loadingModels" style="padding:24px 0;text-align:center;color:#909399">加载模型列表中…</div>
        <div v-else-if="!groupedModels.length" style="padding:24px 0;text-align:center;color:#909399">暂无可用模型</div>
        <div v-else style="max-height:380px;overflow:auto;border:1px solid #eee;border-radius:4px">
          <n-collapse :default-expanded-names="defaultExpanded" arrow-placement="left">
            <n-collapse-item v-for="g in groupedModels" :key="g.provider" :name="g.provider">
              <!-- 分组标题：全选 checkbox + provider 名 + 计数 -->
              <template #header>
                <div style="display:flex;align-items:center;gap:8px;width:100%" @click.stop>
                  <n-checkbox
                    :checked="groupChecked(g)"
                    :indeterminate="groupIndeterminate(g)"
                    @update:checked="(v) => toggleGroup(g, v)"
                  />
                  <span style="font-weight:600">{{ g.provider }}</span>
                  <span style="font-size:12px;color:#909399">{{ groupSelectedCount(g) }}/{{ g.models.length }}</span>
                </div>
              </template>
              <!-- 模型多列网格 -->
              <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:6px 12px;padding:4px 0 4px 24px">
                <n-checkbox
                  v-for="m in g.models"
                  :key="m.id"
                  :checked="form.modelIds.includes(m.id)"
                  @update:checked="(v) => toggleOne(m.id, v)"
                >
                  <span :title="m.model">{{ m.model }}</span>
                </n-checkbox>
              </div>
            </n-collapse-item>
          </n-collapse>
        </div>
      </template>
    </div>
    <p v-if="errMsg" style="color:#d03050;font-size:13px;margin-top:8px">{{ errMsg }}</p>
    <template #footer>
      <n-space justify="end">
        <n-button @click="show=false">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="doSubmit">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { NModal, NRadioGroup, NRadio, NCheckbox, NSpace, NButton, NInput, NCollapse, NCollapseItem, useMessage } from 'naive-ui'
import {
  getApiKeyModelAccess,
  setApiKeyModelAccess,
  getApiKeyModelAccessSelf,
  setApiKeyModelAccessSelf,
} from '../api'

const props = defineProps({
  visible: { type: Boolean, default: false },
  apiKeyId: { type: Number, default: 0 },
  title: { type: String, default: '模型访问权限' },
  admin: { type: Boolean, default: false },
  models: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:visible', 'saved'])

const message = useMessage()
const show = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v),
})

const loadingModels = ref(false)
const submitting = ref(false)
const errMsg = ref('')
const keyword = ref('')
const form = ref({ policy: 'all', modelIds: [] })

// 按 provider 分组，并按关键词过滤
const groupedModels = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const map = {}
  for (const m of props.models) {
    if (kw && !String(m.model).toLowerCase().includes(kw)) continue
    if (!map[m.provider]) map[m.provider] = []
    map[m.provider].push(m)
  }
  return Object.keys(map).sort().map(k => ({ provider: k, models: map[k] }))
})

// 默认展开所有分组
const defaultExpanded = computed(() => groupedModels.value.map(g => g.provider))

const totalCount = computed(() => props.models.length)
const selectedCount = computed(() => form.value.modelIds.length)

// 分组选中状态
function groupSelectedCount(g) {
  return g.models.filter(m => form.value.modelIds.includes(m.id)).length
}
function groupChecked(g) {
  return g.models.length > 0 && g.models.every(m => form.value.modelIds.includes(m.id))
}
function groupIndeterminate(g) {
  const c = groupSelectedCount(g)
  return c > 0 && c < g.models.length
}
function toggleGroup(g, checked) {
  const ids = g.models.map(m => m.id)
  if (checked) {
    const set = new Set(form.value.modelIds)
    ids.forEach(id => set.add(id))
    form.value.modelIds = [...set]
  } else {
    const remove = new Set(ids)
    form.value.modelIds = form.value.modelIds.filter(id => !remove.has(id))
  }
}
function toggleOne(id, checked) {
  if (checked) {
    if (!form.value.modelIds.includes(id)) {
      form.value.modelIds = [...form.value.modelIds, id]
    }
  } else {
    form.value.modelIds = form.value.modelIds.filter(x => x !== id)
  }
}
function selectAllFiltered() {
  const set = new Set(form.value.modelIds)
  groupedModels.value.forEach(g => g.models.forEach(m => set.add(m.id)))
  form.value.modelIds = [...set]
}
function clearAll() {
  form.value.modelIds = []
}

// 弹窗打开时加载当前配置
watch(show, async (v) => {
  if (!v) return
  errMsg.value = ''
  keyword.value = ''
  form.value = { policy: 'all', modelIds: [] }
  if (!props.apiKeyId) return
  loadingModels.value = true
  try {
    const getter = props.admin ? getApiKeyModelAccess : getApiKeyModelAccessSelf
    const data = await getter(props.apiKeyId)
    form.value.policy = data.model_policy || 'all'
    form.value.modelIds = data.model_ids || []
  } catch (e) {
    errMsg.value = e.msg || '加载失败'
  } finally {
    loadingModels.value = false
  }
})

async function doSubmit() {
  errMsg.value = ''
  submitting.value = true
  try {
    const setter = props.admin ? setApiKeyModelAccess : setApiKeyModelAccessSelf
    await setter(props.apiKeyId, form.value.policy, form.value.policy === 'whitelist' ? form.value.modelIds : [])
    message.success('已保存')
    show.value = false
    emit('saved')
  } catch (e) {
    errMsg.value = e.msg || '保存失败'
  } finally {
    submitting.value = false
  }
}
</script>
