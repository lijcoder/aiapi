<template>
  <div>
    <n-card size="small" :bordered="false" style="background:#fff;margin-bottom:12px">
      <div style="display:flex;align-items:center;gap:12px;font-size:14px">
        <span style="color:#666">当前余额</span>
        <span style="font-size:22px;font-weight:700;color:#e53935">¥ {{ fix4(balance) }}</span>
        <span style="color:#999;font-size:12px">充值由管理员发起，如需充值请联系管理员</span>
      </div>
    </n-card>
    <n-card title="充值记录" size="small">
      <n-data-table :columns="columns" :data="records" :loading="tableLoading" :bordered="false" size="small" :pagination="pagination" :remote="true" @update:page="onPage" @update:page-size="onPageSize" style="width:100%" />
    </n-card>
  </div>
</template>

<script setup>
import { ref, onMounted, h } from 'vue'
import { NCard, NDataTable } from 'naive-ui'
import { useUser } from '../stores/user'
import { rechargeSelfRecords } from '../api'
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
