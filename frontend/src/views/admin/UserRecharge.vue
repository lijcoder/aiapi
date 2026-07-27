<template>
  <div>
    <n-card size="small">
      <template #header>
        <n-breadcrumb>
          <n-breadcrumb-item @click="goBack">用户管理</n-breadcrumb-item>
          <n-breadcrumb-item>{{ userName }} 的充值记录</n-breadcrumb-item>
        </n-breadcrumb>
      </template>
      <template #header-extra>
        <n-button size="small" @click="goBack">返回</n-button>
      </template>
      <n-data-table
        :columns="columns"
        :data="records"
        :loading="tableLoading"
        :bordered="false"
        size="small"
        :scroll-x="900"
        :pagination="pagination"
        :remote="true"
        @update:page="onPage"
        style="width:100%"
      />
    </n-card>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NDataTable, NButton, NBreadcrumb, NBreadcrumbItem, useMessage } from 'naive-ui'
import { rechargeRecords, listUsers } from '../../api'
import { fix4, formatTime } from '../../utils'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const userId = Number(route.params.id)
const userName = ref('')

const records = ref([])
const tableLoading = ref(false)
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: false })

function onPage(p) { pagination.value.page = p; loadRecords() }

const columns = [
  { title: '金额', key: 'amount', width: 120, align: 'center', render(r) { return h('span', {style:'color:#18a058;font-weight:600'}, '¥ ' + fix4(r.amount)) }},
  { title: '充值前', key: 'balance_before', width: 120, align: 'center', render(r) { return '¥ ' + fix4(r.balance_before) }},
  { title: '充值后', key: 'balance_after', width: 120, align: 'center', render(r) { return '¥ ' + fix4(r.balance_after) }},
  { title: '操作人', key: 'operator_name', width: 100, align: 'center', render(r) { return r.operator_name || r.operator || '-' }},
  { title: '时间', key: 'created_at', width: 170, align: 'center', ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) }},
  { title: '备注', key: 'remark', align: 'center', ellipsis: { tooltip: true } },
]

function goBack() {
  router.push('/admin/users')
}

async function loadUserInfo() {
  try {
    const data = await listUsers('')
    const u = (data?.users || []).find(x => x.id === userId)
    if (u) userName.value = u.name || u.account
  } catch {}
}

async function loadRecords() {
  tableLoading.value = true
  try {
    const res = await rechargeRecords(userId, pagination.value.page, pagination.value.pageSize)
    records.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

onMounted(() => {
  loadUserInfo()
  loadRecords()
})
</script>
