<template>
  <div>
    <n-card size="small" :bordered="false" style="background:#fff;margin-bottom:12px">
      <div style="display:flex;align-items:center;gap:12px;font-size:14px">
        <span style="color:#666">当前余额</span>
        <span style="font-size:22px;font-weight:700;color:#e53935">¥ {{ fix4(balance) }}</span>
      </div>
    </n-card>
    <n-card title="充值流水" size="small">
      <template #header-extra>
        <n-button type="primary" size="small" @click="openDialog">充值</n-button>
      </template>
      <n-data-table :columns="columns" :data="records" :loading="tableLoading" :bordered="false" size="small" :pagination="pagination" :remote="true" @update:page="onPage" @update:page-size="onPageSize" style="width:100%" />
    </n-card>

    <n-modal v-model:show="showDialog" preset="card" title="账户充值" style="width:400px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <n-input-number v-model:value="amount" :min="1" :step="1" placeholder="金额" style="width:100%" />
        <n-input v-model:value="remark" placeholder="备注（选填）" />
      </div>
      <p v-if="msg" :style="{color:ok?'#18a058':'#d03050',fontSize:'13px',marginTop:'8px'}">{{ msg }}</p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="closeDialog">取消</n-button>
          <n-button type="primary" :loading="loading" @click="doRecharge">确认充值</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, h } from 'vue'
import { NCard, NDataTable, NModal, NInputNumber, NInput, NButton, NSpace } from 'naive-ui'
import { useUser } from '../stores/user'
import { rechargeSelf, rechargeSelfRecords } from '../api'
import { usePagination } from '../composables/usePagination'
import { fix4, formatTime } from '../utils'

const { user, fetchUser } = useUser()

const balance = ref(0)

const columns = [
  { title: '金额', key: 'amount', width: 120, render(r) { return h('span', {style:'color:#18a058;font-weight:600'}, '¥ ' + fix4(r.amount)) }},
  { title: '充值前', key: 'balance_before', width: 120, render(r) { return '¥ ' + fix4(r.balance_before) }},
  { title: '充值后', key: 'balance_after', width: 120, render(r) { return '¥ ' + fix4(r.balance_after) }},
  { title: '操作人', key: 'operator_name', width: 100, render(r) { return r.operator_name || r.operator || '-' }},
  { title: '时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) }},
  { title: '备注', key: 'remark', ellipsis: { tooltip: true } },
]

const showDialog = ref(false)
const amount = ref(1)
const remark = ref('')
const loading = ref(false)
const msg = ref('')
const ok = ref(true)

function openDialog() { showDialog.value = true }
function closeDialog() { showDialog.value = false; amount.value = 1; remark.value = ''; msg.value = '' }

async function doRecharge() {
  msg.value = ''
  if (!amount.value || amount.value <= 0) return
  loading.value = true
  try {
    await rechargeSelf(amount.value, remark.value)
    await fetchUser()
    await loadRecords()
    closeDialog()
  } catch (e) {
    msg.value = e.msg || '充值失败'; ok.value = false
  } finally { loading.value = false }
}

const records = ref([])
const tableLoading = ref(false)
const { pagination, onPage, onPageSize } = usePagination(loadRecords)

async function loadRecords() {
  tableLoading.value = true
  try {
    const res = await rechargeSelfRecords(pagination.value.page, pagination.value.pageSize)
    records.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
    await fetchUser()
    balance.value = user.value?.budget || 0
  } catch {} finally { tableLoading.value = false }
}

onMounted(() => loadRecords())
</script>
