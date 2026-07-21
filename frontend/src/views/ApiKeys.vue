<template>
  <div>
    <n-card title="API 密钥" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="openCreate">创建密钥</n-button>
      </template>
      <n-data-table :columns="columns" :data="keys" :loading="tableLoading" :bordered="false" size="small" style="width:100%" />
    </n-card>

    <!-- 创建弹窗 -->
    <n-modal v-model:show="showCreate" preset="card" title="创建 API 密钥" style="width:460px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">名称（选填）</div>
          <n-input v-model:value="form.name" placeholder="例如：测试环境" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">额度模式</div>
          <n-radio-group v-model:value="form.mode">
            <n-space>
              <n-radio value="limited">限制（使用独立额度）</n-radio>
              <n-radio value="unlimited">无限制（共享账户额度）</n-radio>
            </n-space>
          </n-radio-group>
        </div>
        <div v-if="form.mode==='limited'">
          <div style="font-size:13px;margin-bottom:6px">额度金额（最多可设 {{ createBudgetMax }}）</div>
          <n-input-number v-model:value="form.budget" :min="0" :step="1" :status="createBudgetOver ? 'error' : undefined" style="width:100%" />
          <div v-if="createBudgetOver" style="font-size:12px;color:#d03050;margin-top:6px">超出剩余可用额度，最多可设 {{ createBudgetMax }}</div>
          <div v-else style="font-size:12px;color:#909399;margin-top:6px">所有「限制」密钥的额度总和不能超过账户余额 ¥ {{ fix4(user?.budget) }}</div>
        </div>
      </div>
      <p v-if="createMsg" :style="{color:'#d03050',fontSize:'13px',marginTop:'8px'}">{{ createMsg }}</p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="closeCreate">取消</n-button>
          <n-button type="primary" :loading="creating" :disabled="createBudgetOver" @click="doCreate">确认创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 创建成功弹窗：明文 key 只展示这一次 -->
    <n-modal v-model:show="showPlain" preset="card" title="密钥创建成功" style="width:560px" :mask-closable="false" :close-on-esc="false">
      <n-alert type="warning" :show-icon="true" style="margin-bottom:12px">
        请立即复制保存，关闭后将无法再次查看完整密钥。
      </n-alert>
      <n-input :value="plainKey" readonly type="text" />
      <template #footer>
        <n-space justify="end">
          <n-button @click="copyPlain">复制</n-button>
          <n-button type="primary" @click="showPlain=false">我已保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 改名弹窗 -->
    <n-modal v-model:show="showRename" preset="card" title="重命名" style="width:400px" :mask-closable="false">
      <n-input v-model:value="renameVal" placeholder="名称" maxlength="64" />
      <template #footer>
        <n-space justify="end">
          <n-button @click="showRename=false">取消</n-button>
          <n-button type="primary" :loading="renaming" @click="doRename">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 修改额度弹窗 -->
    <n-modal v-model:show="showBudget" preset="card" title="修改额度" style="width:440px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <n-radio-group v-model:value="budgetMode">
          <n-space>
            <n-radio value="limited">限制（使用独立额度）</n-radio>
            <n-radio value="unlimited">无限制（共享账户额度）</n-radio>
          </n-space>
        </n-radio-group>
        <div v-if="budgetMode==='limited'">
          <div style="font-size:13px;margin-bottom:6px">额度金额（最多可设 {{ budgetMax }}）</div>
          <n-input-number v-model:value="budgetVal" :min="0" :step="1" :status="budgetOver ? 'error' : undefined" style="width:100%" />
          <div v-if="budgetOver" style="font-size:12px;color:#d03050;margin-top:6px">超出剩余可用额度，最多可设 {{ budgetMax }}</div>
          <div v-else style="font-size:12px;color:#909399;margin-top:6px">所有「限制」密钥的额度总和不能超过账户余额 ¥ {{ fix4(user?.budget) }}</div>
        </div>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showBudget=false">取消</n-button>
          <n-button type="primary" :loading="savingBudget" :disabled="budgetOver" @click="doSaveBudget">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onMounted, computed } from 'vue'
import { NCard, NDataTable, NModal, NInput, NInputNumber, NButton, NSpace, NAlert, NTag, NIcon, NTooltip, NRadioGroup, NRadio, useMessage } from 'naive-ui'
import { CreateOutline } from '@vicons/ionicons5'
import { listMyApiKeys, createApiKey, toggleApiKey, deleteApiKey, renameApiKey, updateApiKeyBudget } from '../api'
import { useUser } from '../stores/user'
import { fix4 } from '../utils'

const message = useMessage()
const { user } = useUser()

const keys = ref([])
const tableLoading = ref(false)

const showCreate = ref(false)
const showPlain = ref(false)
const showRename = ref(false)
const plainKey = ref('')
const creating = ref(false)
const createMsg = ref('')
const form = ref({ name: '', budget: 0, mode: 'limited' })

// 创建时可用最大额度 = 用户余额 - 现有所有有限额 key 的 budget 之和
const createBudgetMax = computed(() => {
  const userBudget = user.value?.budget ?? 0
  const othersSum = keys.value
    .filter(k => !k.unlimited)
    .reduce((s, k) => s + (Number(k.budget) || 0), 0)
  return Math.max(0, Number((userBudget - othersSum).toFixed(4)))
})

// 创建表单限制模式下是否超额（实时校验）
const createBudgetOver = computed(() => {
  if (form.value.mode !== 'limited') return false
  if (form.value.budget == null) return false
  return form.value.budget > createBudgetMax.value
})

const renameId = ref(0)
const renameVal = ref('')
const renaming = ref(false)

// 修改额度弹窗
const showBudget = ref(false)
const budgetId = ref(0)
const budgetMode = ref('limited')   // 'limited' | 'unlimited'
const budgetVal = ref(0)
const budgetMax = ref(0)            // 当前 key 可设的最大值 = 用户余额 - 其它有限额 key 之和
const savingBudget = ref(false)

// 限制模式下是否超额（实时校验，用于输入框变红 + 禁用保存）
const budgetOver = computed(() => {
  if (budgetMode.value !== 'limited') return false
  if (budgetVal.value == null) return false
  return budgetVal.value > budgetMax.value
})

const columns = [
  {
    title: '名称', key: 'name', width: 200, ellipsis: { tooltip: true },
    render(r) {
      const editIcon = h(NIcon, {
        size: 15, color: '#1677ff', style: 'cursor:pointer;vertical-align:middle',
        onClick: (e) => { e.stopPropagation(); openRename(r) }
      }, () => h(CreateOutline))
      const tip = h(NTooltip, null, { trigger: () => editIcon, default: () => '改名' })
      return h('div', { style: 'display:flex;align-items:center;gap:6px' }, [
        tip,
        h('span', { style: 'flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap' }, r.name || '-')
      ])
    }
  },
  { title: 'Key', key: 'key', width: 200, render(r){ return h('code', { style:'font-size:12px;color:#555' }, r.key) } },
  { title: '额度', key: 'budget', width: 110, render(r){ return r.unlimited ? h(NTag,{size:'small',type:'success'},{default:()=>'无限'}) : '¥ ' + fix4(r.budget) } },
  { title: '状态', key: 'enabled', width: 90, render(r){ return r.enabled ? h(NTag,{size:'small',type:'success'},{default:()=>'启用'}) : h(NTag,{size:'small',type:'error'},{default:()=>'禁用'}) } },
  { title: '创建时间', key: 'created_at', width: 170 },
  {
    title: '操作', key: 'actions', width: 210, fixed: 'right',
    render(r) {
      return h(NSpace, { size: 8 }, () => [
        h(NButton, { size: 'small', tertiary: true, type: 'info', onClick: () => openBudget(r) }, () => '修改额度'),
        h(NButton, {
          size: 'small', tertiary: true, type: r.enabled ? 'warning' : 'success',
          onClick: () => onToggle(r)
        }, () => r.enabled ? '禁用' : '启用'),
        h(NButton, { size: 'small', tertiary: true, type: 'error', onClick: () => onDelete(r) }, () => '删除'),
      ])
    }
  }
]

function openCreate() {
  form.value = { name: '', budget: 0, mode: 'limited' }
  createMsg.value = ''
  showCreate.value = true
}
function closeCreate() {
  showCreate.value = false
}

async function doCreate() {
  createMsg.value = ''
  if (form.value.mode === 'limited' && (form.value.budget == null || form.value.budget < 0)) {
    createMsg.value = '金额不能小于 0'
    return
  }
  if (createBudgetOver.value) {
    createMsg.value = `超出剩余可用额度，最多可设 ${createBudgetMax.value}`
    return
  }
  creating.value = true
  try {
    const data = await createApiKey(
      form.value.name,
      form.value.mode === 'limited' ? (form.value.budget || 0) : 0,
      form.value.mode === 'unlimited'
    )
    plainKey.value = data.apiKey.key
    showCreate.value = false
    showPlain.value = true
    await loadKeys()
  } catch (e) {
    createMsg.value = e.msg || '创建失败'
  } finally {
    creating.value = false
  }
}

async function copyPlain() {
  try {
    await navigator.clipboard.writeText(plainKey.value)
    message.success('已复制')
  } catch {
    message.error('复制失败，请手动选中复制')
  }
}

async function onToggle(r) {
  try {
    await toggleApiKey(r.id)
    await loadKeys()
    message.success(r.enabled ? '已禁用' : '已启用')
  } catch (e) {
    message.error(e.msg || '操作失败')
  }
}

function onDelete(r) {
  const dialog = window.confirm(`确定删除密钥「${r.name || r.key}」吗？此操作不可恢复。`)
  if (!dialog) return
  deleteApiKey(r.id).then(() => {
    message.success('已删除')
    loadKeys()
  }).catch(e => {
    message.error(e.msg || '删除失败')
  })
}

function openRename(r) {
  renameId.value = r.id
  renameVal.value = r.name || ''
  showRename.value = true
}

async function doRename() {
  if (!renameVal.value.trim()) {
    message.warning('名称不能为空')
    return
  }
  renaming.value = true
  try {
    await renameApiKey(renameId.value, renameVal.value.trim())
    showRename.value = false
    await loadKeys()
    message.success('已保存')
  } catch (e) {
    message.error(e.msg || '保存失败')
  } finally {
    renaming.value = false
  }
}

function openBudget(r) {
  budgetId.value = r.id
  budgetMode.value = r.unlimited ? 'unlimited' : 'limited'
  budgetVal.value = Number(r.budget) || 0
  // 计算该 key 可设的最大值：用户余额 - 其它有限额 key 的 budget 之和
  const userBudget = user.value?.budget ?? 0
  const othersSum = keys.value
    .filter(k => k.id !== r.id && !k.unlimited)
    .reduce((s, k) => s + (Number(k.budget) || 0), 0)
  budgetMax.value = Math.max(0, Number((userBudget - othersSum).toFixed(4)))
  showBudget.value = true
}

async function doSaveBudget() {
  if (budgetMode.value === 'limited' && (budgetVal.value == null || budgetVal.value < 0)) {
    message.warning('金额不能小于 0')
    return
  }
  if (budgetOver.value) {
    // 兑底，正常情况按钮已禁用
    message.warning(`超出剩余可用额度，最多可设 ${budgetMax.value}`)
    return
  }
  savingBudget.value = true
  try {
    await updateApiKeyBudget(
      budgetId.value,
      budgetMode.value === 'limited' ? budgetVal.value : 0,
      budgetMode.value === 'unlimited'
    )
    showBudget.value = false
    await loadKeys()
    message.success('已保存')
  } catch (e) {
    message.error(e.msg || '保存失败')
  } finally {
    savingBudget.value = false
  }
}

async function loadKeys() {
  tableLoading.value = true
  try { keys.value = (await listMyApiKeys()) || [] } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

onMounted(() => loadKeys())
</script>
