<template>
  <div>
    <n-card size="small">
      <template #header>
        <n-breadcrumb>
          <n-breadcrumb-item @click="goBack">用户管理</n-breadcrumb-item>
          <n-breadcrumb-item>{{ userName }} 的 API Key</n-breadcrumb-item>
        </n-breadcrumb>
      </template>
      <template #header-extra>
        <n-button size="small" @click="goBack">返回</n-button>
      </template>
      <n-data-table :columns="columns" :data="keys" :loading="tableLoading" :bordered="false" size="small" :scroll-x="960" :pagination="pagination" :remote="true" @update:page="onPage" @update:page-size="onPageSize" style="width:100%" />
    </n-card>

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
          <div v-else style="font-size:12px;color:#909399;margin-top:6px">所有「限制」密钥的额度总和不能超过账户余额 ¥ {{ fix4(userBudget) }}</div>
        </div>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showBudget=false">取消</n-button>
          <n-button type="primary" :loading="savingBudget" :disabled="budgetOver" @click="doSaveBudget">保存</n-button>
        </n-space>
      </template>
    </n-modal>
    <!-- 模型权限弹窗 -->
    <ModelAccessDialog
      v-model:visible="showModelAccess"
      :api-key-id="modelAccessKeyId"
      :admin="true"
      title="模型访问权限"
      @saved="loadKeys"
    />
  </div>
</template>

<script setup>
import { ref, h, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NDataTable, NModal, NInput, NInputNumber, NButton, NSpace, NTag, NIcon, NTooltip, NDropdown, NRadioGroup, NRadio, NBreadcrumb, NBreadcrumbItem, useMessage, useDialog } from 'naive-ui'
import { CreateOutline } from '@vicons/ionicons5'
import { listUserApiKeys, toggleUserApiKey, deleteUserApiKey, renameUserApiKey, updateUserApiKeyBudget, getUser } from '../../api'
import { usePagination } from '../../composables/usePagination'
import ModelAccessDialog from '../../components/ModelAccessDialog.vue'
import { fix4, formatTime } from '../../utils'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const userId = Number(route.params.id)
const userName = ref('')
const userBudget = ref(0)

const keys = ref([])
const tableLoading = ref(false)
const { pagination, onPage, onPageSize } = usePagination(loadKeys)

const showRename = ref(false)
const renameId = ref(0)
const renameVal = ref('')
const renaming = ref(false)

const showBudget = ref(false)
const budgetId = ref(0)
const budgetMode = ref('limited')
const budgetVal = ref(0)
const budgetMax = ref(0)
const savingBudget = ref(false)

// 模型权限弹窗
const showModelAccess = ref(false)
const modelAccessKeyId = ref(0)

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
  { title: '创建时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) } },
  {
    title: '操作', key: 'actions', width: 150, fixed: 'right',
    render(r) {
      const moreOptions = [
        { label: '模型权限', key: 'model_access' },
        { label: '修改额度', key: 'budget' },
        { type: 'divider', key: 'd1' },
        { label: '删除', key: 'delete' },
      ]
      function onSelect(key) {
        if (key === 'model_access') openModelAccess(r)
        else if (key === 'budget') openBudget(r)
        else if (key === 'delete') onDelete(r)
      }
      return h(NSpace, { size: 6 }, () => [
        h(NButton, {
          size: 'small', tertiary: true, type: r.enabled ? 'warning' : 'success',
          onClick: () => onToggle(r)
        }, () => r.enabled ? '禁用' : '启用'),
        h(NDropdown, { options: moreOptions, trigger: 'click', onSelect: (k) => onSelect(k) }, {
          default: () => h(NButton, { size: 'small', tertiary: true }, () => '更多')
        }),
      ])
    }
  }
]

function goBack() {
  router.push('/admin/users')
}

async function loadUserInfo() {
  try {
    const u = await getUser(userId)
    if (u) {
      userName.value = u.name || u.account
      userBudget.value = u.unlimited ? 0 : (u.budget || 0)
    }
  } catch {}
}

async function loadKeys() {
  tableLoading.value = true
  try {
    const res = await listUserApiKeys(userId, pagination.value.page, pagination.value.pageSize)
    keys.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

async function onToggle(r) {
  try {
    await toggleUserApiKey(r.id)
    await loadKeys()
    message.success(r.enabled ? '已禁用' : '已启用')
  } catch (e) {
    message.error(e.msg || '操作失败')
  }
}

function onDelete(r) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除密钥「${r.name || r.key}」吗？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () => {
      deleteUserApiKey(r.id).then(() => {
        message.success('已删除')
        loadKeys()
      }).catch(e => {
        message.error(e.msg || '删除失败')
      })
    }
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
    await renameUserApiKey(renameId.value, renameVal.value.trim())
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
  const othersSum = keys.value
    .filter(k => k.id !== r.id && !k.unlimited)
    .reduce((s, k) => s + (Number(k.budget) || 0), 0)
  budgetMax.value = Math.max(0, Number((userBudget.value - othersSum).toFixed(4)))
  showBudget.value = true
}

function openModelAccess(r) {
  modelAccessKeyId.value = r.id
  showModelAccess.value = true
}

async function doSaveBudget() {
  if (budgetMode.value === 'limited' && (budgetVal.value == null || budgetVal.value < 0)) {
    message.warning('金额不能小于 0')
    return
  }
  if (budgetOver.value) {
    message.warning(`超出剩余可用额度，最多可设 ${budgetMax.value}`)
    return
  }
  savingBudget.value = true
  try {
    await updateUserApiKeyBudget(
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

onMounted(() => {
  loadUserInfo()
  loadKeys()
})
</script>
