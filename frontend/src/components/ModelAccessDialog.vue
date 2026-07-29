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
          搜索并勾选该 Key 允许访问的模型，不勾选任何模型则该 Key 无法访问任何模型
        </div>

        <!-- 已选模型：常驻展示，与搜索结果解耦 -->
        <div style="display:flex;align-items:center;gap:10px">
          <span style="font-size:13px;color:#666">
            已选 <b style="color:#1677ff">{{ selectedCount }}</b> 个
          </span>
          <n-button size="tiny" quaternary :disabled="!selectedCount" @click="clearAll">清空</n-button>
        </div>
        <div v-if="selectedModels.length" style="display:flex;flex-wrap:wrap;gap:6px;max-height:120px;overflow:auto">
          <n-tag v-for="m in selectedModels" :key="m.id" size="small" closable @close="removeOne(m.id)">
            {{ m.model }}<span style="color:#909399;font-size:12px">&nbsp;· {{ m.provider }}</span>
          </n-tag>
        </div>

        <!-- 搜索框：300ms 防抖，服务端模糊查询 -->
        <n-input v-model:value="keyword" placeholder="输入模型名模糊搜索" size="small" clearable />

        <!-- 候选列表 -->
        <div v-if="searching" style="padding:24px 0;text-align:center;color:#909399">搜索中…</div>
        <div v-else-if="!candidates.length" style="padding:24px 0;text-align:center;color:#909399">无匹配模型</div>
        <template v-else>
          <div style="max-height:300px;overflow:auto;border:1px solid #eee;border-radius:4px;padding:8px 12px;display:flex;flex-direction:column;gap:6px">
            <n-checkbox
              v-for="m in candidates"
              :key="m.id"
              :checked="form.modelIds.includes(m.id)"
              @update:checked="(v) => toggleOne(m, v)"
            >
              <span :title="m.model">{{ m.model }}</span>
              <span style="font-size:12px;color:#909399">&nbsp;{{ m.provider }}</span>
            </n-checkbox>
          </div>
          <div style="display:flex;align-items:center;gap:10px">
            <span style="font-size:12px;color:#909399">
              {{ total > pageSize ? `匹配 ${total} 个，仅显示前 ${pageSize} 个，请继续输入精确查找` : `共 ${total} 个` }}
            </span>
            <n-button size="tiny" quaternary style="margin-left:auto" @click="selectAllFiltered">全选当前结果</n-button>
          </div>
        </template>
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
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { NModal, NRadioGroup, NRadio, NCheckbox, NSpace, NButton, NInput, NTag, useMessage } from 'naive-ui'
import {
  getApiKeyModelAccess,
  setApiKeyModelAccess,
  getApiKeyModelAccessSelf,
  setApiKeyModelAccessSelf,
  listModels,
} from '../api'

const props = defineProps({
  visible: { type: Boolean, default: false },
  apiKeyId: { type: Number, default: 0 },
  title: { type: String, default: '模型访问权限' },
  admin: { type: Boolean, default: false },
})

const emit = defineEmits(['update:visible', 'saved'])

const message = useMessage()
const show = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v),
})

// 候选列表每页条数：只展示前 N 条匹配结果，更多通过搜索收敛
const pageSize = 10

const loading = ref(false)
const searching = ref(false)
const submitting = ref(false)
const errMsg = ref('')
const keyword = ref('')
const form = ref({ policy: 'all', modelIds: [] })
const selectedMap = ref({}) // id -> {id, provider, model}，已选模型详情（常驻展示用）
const candidates = ref([])  // 当前搜索结果的候选模型
const total = ref(0)        // 当前关键词的匹配总数

// 已选模型按选择顺序展示
const selectedModels = computed(() => form.value.modelIds.map(id => selectedMap.value[id]).filter(Boolean))
const selectedCount = computed(() => form.value.modelIds.length)

// 关键词防抖搜索
let timer = null
watch(keyword, () => {
  clearTimeout(timer)
  timer = setTimeout(searchModels, 300)
})
onBeforeUnmount(() => clearTimeout(timer))

async function searchModels() {
  searching.value = true
  try {
    const res = await listModels('', keyword.value.trim(), 1, pageSize)
    candidates.value = res?.items || []
    total.value = res?.total || 0
  } catch {
    candidates.value = []
    total.value = 0
  } finally {
    searching.value = false
  }
}

// 弹窗打开时加载当前配置 + 首批候选模型
watch(show, async (v) => {
  if (!v) return
  errMsg.value = ''
  keyword.value = ''
  form.value = { policy: 'all', modelIds: [] }
  selectedMap.value = {}
  candidates.value = []
  total.value = 0
  searchModels()
  if (!props.apiKeyId) return
  loading.value = true
  try {
    const getter = props.admin ? getApiKeyModelAccess : getApiKeyModelAccessSelf
    const data = await getter(props.apiKeyId)
    form.value.policy = data.model_policy || 'all'
    form.value.modelIds = data.model_ids || []
    const map = {}
    for (const m of data.models || []) map[m.id] = m
    selectedMap.value = map
  } catch (e) {
    errMsg.value = e.msg || '加载失败'
  } finally {
    loading.value = false
  }
})

function toggleOne(m, checked) {
  if (checked) {
    if (!form.value.modelIds.includes(m.id)) {
      form.value.modelIds = [...form.value.modelIds, m.id]
      selectedMap.value = { ...selectedMap.value, [m.id]: m }
    }
  } else {
    removeOne(m.id)
  }
}
function removeOne(id) {
  form.value.modelIds = form.value.modelIds.filter(x => x !== id)
}
function selectAllFiltered() {
  const set = new Set(form.value.modelIds)
  const map = { ...selectedMap.value }
  for (const m of candidates.value) {
    set.add(m.id)
    map[m.id] = m
  }
  form.value.modelIds = [...set]
  selectedMap.value = map
}
function clearAll() {
  form.value.modelIds = []
}

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
