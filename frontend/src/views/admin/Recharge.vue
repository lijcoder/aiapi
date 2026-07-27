<template>
  <n-card title="充值记录" size="small">
    <template #header-extra>
      <n-space align="center">
        <n-input v-model:value="keyword" placeholder="搜索用户/账号/备注" size="small" clearable style="width:220px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
        <n-button size="small" @click="resetAndLoad">查询</n-button>
      </n-space>
    </template>
    <n-data-table
      :columns="columns"
      :data="records"
      :loading="tableLoading"
      :bordered="false"
      size="small"
      :scroll-x="1000"
      :pagination="pagination"
      :remote="true"
      @update:page="onPage"
      style="width:100%"
    />
  </n-card>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { listAllRechargeRecords } from '../../api'
import { fix4, formatTime } from '../../utils'

const message = useMessage()

const records = ref([])
const keyword = ref('')
const tableLoading = ref(false)
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: false })

function onPage(p) { pagination.value.page = p; load() }
function resetAndLoad() { pagination.value.page = 1; load() }

const columns = [
  { title: 'ID', key: 'id', width: 70, align: 'center' },
  { title: '用户', key: 'user_name', width: 120, align: 'center', ellipsis: { tooltip: true }, render(r) { return r.user_name || r.user_id } },
  { title: '金额', key: 'amount', width: 120, align: 'center', render(r) { return h('span', {style:'color:#18a058;font-weight:600'}, '¥ ' + fix4(r.amount)) }},
  { title: '充值前', key: 'balance_before', width: 120, align: 'center', render(r) { return '¥ ' + fix4(r.balance_before) }},
  { title: '充值后', key: 'balance_after', width: 120, align: 'center', render(r) { return '¥ ' + fix4(r.balance_after) }},
  { title: '操作人', key: 'operator_name', width: 100, align: 'center', render(r) { return r.operator_name || r.operator || '-' }},
  { title: '时间', key: 'created_at', width: 170, align: 'center', ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) }},
  { title: '备注', key: 'remark', align: 'center', ellipsis: { tooltip: true } },
]

async function load() {
  tableLoading.value = true
  try {
    const res = await listAllRechargeRecords(keyword.value, pagination.value.page, pagination.value.pageSize)
    records.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

onMounted(() => load())
</script>
