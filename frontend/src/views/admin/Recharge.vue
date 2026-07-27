<template>
  <n-card :title="title" size="small">
    <template #header-extra>
      <n-space align="center">
        <template v-if="userId">
          <n-button size="small" @click="goBack">返回</n-button>
        </template>
        <template v-else>
          <n-input v-model:value="keyword" placeholder="搜索用户/账号/备注" size="small" clearable style="width:220px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
          <n-button size="small" @click="resetAndLoad">查询</n-button>
        </template>
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
import { ref, computed, h, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NDataTable, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { listAllRechargeRecords, rechargeRecords, getUser } from '../../api'
import { fix4, formatTime } from '../../utils'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const userId = computed(() => Number(route.params.id) || 0)
const userName = ref('')

const title = computed(() => userId.value ? `${userName.value || ('用户 #' + userId.value)} 的充值记录` : '充值记录')

const records = ref([])
const keyword = ref('')
const tableLoading = ref(false)
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: false })

function onPage(p) { pagination.value.page = p; load() }
function resetAndLoad() { pagination.value.page = 1; load() }
function goBack() { router.push('/admin/users') }

const columns = computed(() => {
  const cols = []
  if (!userId.value) {
    cols.push({ title: 'ID', key: 'id', width: 70 })
    cols.push({ title: '用户', key: 'user_name', width: 120, ellipsis: { tooltip: true }, render(r) { return r.user_name || r.user_id } })
  }
  cols.push({ title: '金额', key: 'amount', width: 120, render(r) { return h('span', {style:'color:#18a058;font-weight:600'}, '¥ ' + fix4(r.amount)) }})
  cols.push({ title: '充值前', key: 'balance_before', width: 120, render(r) { return '¥ ' + fix4(r.balance_before) }})
  cols.push({ title: '充值后', key: 'balance_after', width: 120, render(r) { return '¥ ' + fix4(r.balance_after) }})
  cols.push({ title: '操作人', key: 'operator_name', width: 100, render(r) { return r.operator_name || r.operator || '-' }})
  cols.push({ title: '时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) }})
  cols.push({ title: '备注', key: 'remark', ellipsis: { tooltip: true } })
  return cols
})

async function loadUserInfo() {
  if (!userId.value) return
  try {
    const u = await getUser(userId.value)
    if (u) userName.value = u.name || u.account
  } catch {}
}

async function load() {
  tableLoading.value = true
  try {
    let res
    if (userId.value) {
      res = await rechargeRecords(userId.value, pagination.value.page, pagination.value.pageSize)
    } else {
      res = await listAllRechargeRecords(keyword.value, pagination.value.page, pagination.value.pageSize)
    }
    records.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

onMounted(() => {
  loadUserInfo()
  load()
})
</script>
