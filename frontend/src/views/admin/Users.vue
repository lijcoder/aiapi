<template>
  <div>
    <n-card title="用户管理" size="small">
      <template #header-extra>
        <n-space align="center">
          <n-input v-model:value="keyword" placeholder="搜索姓名/账号" size="small" clearable style="width:200px" @keydown.enter="resetAndLoad" @clear="resetAndLoad" />
          <n-button size="small" type="primary" @click="openCreate">创建用户</n-button>
        </n-space>
      </template>
      <n-data-table
        :columns="columns"
        :data="users"
        :loading="tableLoading"
        :bordered="false"
        size="small"
        :scroll-x="1000"
        :pagination="pagination"
        :remote="true"
        @update:page="onPage"
        @update:page-size="onPageSize"
        style="width:100%"
      />
    </n-card>

    <!-- 创建用户弹窗 -->
    <n-modal v-model:show="showCreate" preset="card" title="创建用户" style="width:460px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">姓名</div>
          <n-input v-model:value="form.name" placeholder="姓名" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">账号</div>
          <n-input v-model:value="form.account" placeholder="登录账号" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">初始密码</div>
          <n-input v-model:value="form.password" type="password" placeholder="初始密码" show-password-on="click" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">额度模式</div>
          <n-radio-group v-model:value="form.mode">
            <n-space>
              <n-radio value="limited">限制</n-radio>
              <n-radio value="unlimited">无限制</n-radio>
            </n-space>
          </n-radio-group>
        </div>
        <div v-if="form.mode==='limited'">
          <div style="font-size:13px;margin-bottom:6px">额度金额</div>
          <n-input-number v-model:value="form.budget" :min="0" :step="1" style="width:100%" />
        </div>
      </div>
      <p v-if="formMsg" style="color:#d03050;font-size:13px;margin-top:8px">{{ formMsg }}</p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreate=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doCreate">确认创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑用户弹窗 -->
    <n-modal v-model:show="showEdit" preset="card" title="编辑用户" style="width:460px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">姓名</div>
          <n-input v-model:value="editForm.name" placeholder="姓名" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">账号（不可修改）</div>
          <n-input :value="editForm.account" disabled />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">额度模式</div>
          <n-radio-group v-model:value="editForm.mode">
            <n-space>
              <n-radio value="limited">限制</n-radio>
              <n-radio value="unlimited">无限制</n-radio>
            </n-space>
          </n-radio-group>
        </div>
        <div v-if="editForm.mode==='limited'">
          <div style="font-size:13px;margin-bottom:6px">额度金额</div>
          <n-input-number v-model:value="editForm.budget" :min="0" :step="1" style="width:100%" />
        </div>
      </div>
      <p v-if="editFormMsg" style="color:#d03050;font-size:13px;margin-top:8px">{{ editFormMsg }}</p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEdit=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doEdit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 重置密码弹窗 -->
    <n-modal v-model:show="showReset" preset="card" title="重置密码" style="width:400px" :mask-closable="false">
      <n-input v-model:value="resetPwd" type="password" placeholder="新密码" show-password-on="click" />
      <template #footer>
        <n-space justify="end">
          <n-button @click="showReset=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doReset">确认</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 分配角色弹窗 -->
    <n-modal v-model:show="showAssign" preset="card" title="分配角色" style="width:400px" :mask-closable="false">
      <n-checkbox-group v-model:value="assignRoleIds">
        <n-space vertical>
          <n-checkbox v-for="r in allRoles" :key="r.id" :value="r.id" :label="r.name" />
        </n-space>
      </n-checkbox-group>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAssign=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doAssign">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 充值弹窗 -->
    <n-modal v-model:show="showRecharge" preset="card" title="充值" style="width:400px" :mask-closable="false">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">金额</div>
          <n-input-number v-model:value="rechargeAmount" :min="1" :step="1" style="width:100%" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">备注（选填）</div>
          <n-input v-model:value="rechargeRemark" placeholder="备注" />
        </div>
      </div>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showRecharge=false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="doRecharge">确认充值</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NCard, NDataTable, NModal, NInput, NInputNumber, NButton, NSpace, NTag, NDropdown, NRadioGroup, NRadio, NCheckboxGroup, NCheckbox, useMessage, useDialog } from 'naive-ui'
import { useRouter } from 'vue-router'
import { listUsers, createUser, updateUser, toggleUser, resetPassword as resetPasswordApi, assignRoles, rechargeAdmin, listRoles } from '../../api'
import { usePagination } from '../../composables/usePagination'
import { fix4, formatTime } from '../../utils'

const message = useMessage()
const dialog = useDialog()
const router = useRouter()

const users = ref([])
const allRoles = ref([])
const keyword = ref('')
const tableLoading = ref(false)
const { pagination, onPage, onPageSize, resetAndLoad } = usePagination(load)

// 创建
const showCreate = ref(false)
const form = ref({ name: '', account: '', password: '', budget: 0, mode: 'limited' })
const formMsg = ref('')

// 编辑
const showEdit = ref(false)
const editForm = ref({ id: 0, name: '', account: '', budget: 0, mode: 'limited' })
const editFormMsg = ref('')

// 重置密码
const showReset = ref(false)
const resetTargetId = ref(0)
const resetPwd = ref('')

// 分配角色
const showAssign = ref(false)
const assignTargetId = ref(0)
const assignRoleIds = ref([])

// 充值
const showRecharge = ref(false)
const rechargeTargetId = ref(0)
const rechargeAmount = ref(1)
const rechargeRemark = ref('')

const submitting = ref(false)

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '姓名', key: 'name', width: 120, ellipsis: { tooltip: true } },
  { title: '账号', key: 'account', width: 140, ellipsis: { tooltip: true } },
  { title: '余额', key: 'budget', width: 110, render(r) { return r.unlimited ? h(NTag, { size:'small', type:'success' }, { default: () => '无限' }) : '¥ ' + fix4(r.budget) } },
  { title: '角色', key: 'role_names', width: 160, render(r) {
    if (!r.role_names || !r.role_names.length) return '-'
    return h(NSpace, { size: 4 }, () => r.role_names.map(n => h(NTag, { size:'small' }, { default: () => n })))
  }},
  { title: '状态', key: 'enabled', width: 80, render(r) {
    return r.enabled ? h(NTag, { size:'small', type:'success' }, { default: () => '启用' }) : h(NTag, { size:'small', type:'error' }, { default: () => '禁用' })
  }},
  { title: '创建时间', key: 'created_at', width: 170, ellipsis: { tooltip: true }, render(r) { return formatTime(r.created_at) } },
  { title: '操作', key: 'actions', width: 160, fixed: 'right', render(r) {
    const moreOptions = [
      { label: '充值', key: 'recharge' },
      { label: '充值记录', key: 'recharge_records' },
      { label: 'API Key', key: 'apikeys' },
      { label: '重置密码', key: 'reset' },
      { label: '分配角色', key: 'assign' },
    ]
    function onSelect(key) {
      if (key === 'reset') openReset(r)
      else if (key === 'assign') openAssign(r)
      else if (key === 'recharge') openRecharge(r)
      else if (key === 'recharge_records') goRechargeRecords(r)
      else if (key === 'apikeys') goApiKeys(r)
    }
    return h(NSpace, { size: 6 }, () => [
      h(NButton, { size: 'small', tertiary: true, type: 'info', onClick: () => openEdit(r) }, () => '编辑'),
      h(NButton, {
        size: 'small', tertiary: true, type: r.enabled ? 'warning' : 'success',
        onClick: () => onToggle(r)
      }, () => r.enabled ? '禁用' : '启用'),
      h(NDropdown, { options: moreOptions, trigger: 'click', onSelect: (k) => onSelect(k) }, {
        default: () => h(NButton, { size: 'small', tertiary: true }, () => '更多')
      }),
    ])
  }},
]

async function loadRoles() {
  try {
    const data = await listRoles()
    allRoles.value = data?.roles || []
  } catch (e) {
    // 静默失败，不阻塞页面
  }
}

async function load() {
  tableLoading.value = true
  try {
    const res = await listUsers(keyword.value, pagination.value.page, pagination.value.pageSize)
    users.value = res?.items || []
    pagination.value.itemCount = res?.total || 0
  } catch (e) {
    message.error(e.msg || '加载失败')
  } finally { tableLoading.value = false }
}

function openCreate() {
  form.value = { name: '', account: '', password: '', budget: 0, mode: 'limited' }
  formMsg.value = ''
  showCreate.value = true
}

async function doCreate() {
  formMsg.value = ''
  if (!form.value.name || !form.value.account || !form.value.password) {
    formMsg.value = '姓名、账号、密码均为必填'
    return
  }
  submitting.value = true
  try {
    await createUser(form.value.name, form.value.account, form.value.password, form.value.budget || 0, form.value.mode === 'unlimited')
    showCreate.value = false
    await load()
    message.success('创建成功')
  } catch (e) {
    formMsg.value = e.msg || '创建失败'
  } finally { submitting.value = false }
}

function openEdit(r) {
  editForm.value = {
    id: r.id,
    name: r.name,
    account: r.account,
    budget: r.budget,
    mode: r.unlimited ? 'unlimited' : 'limited'
  }
  editFormMsg.value = ''
  showEdit.value = true
}

async function doEdit() {
  editFormMsg.value = ''
  if (!editForm.value.name) {
    editFormMsg.value = '姓名不能为空'
    return
  }
  submitting.value = true
  try {
    await updateUser(editForm.value.id, editForm.value.name, editForm.value.budget || 0, editForm.value.mode === 'unlimited')
    showEdit.value = false
    await load()
    message.success('已保存')
  } catch (e) {
    editFormMsg.value = e.msg || '保存失败'
  } finally { submitting.value = false }
}

async function onToggle(r) {
  if (r.enabled) {
    // 禁用前确认
    dialog.warning({
      title: '确认禁用',
      content: `确定禁用用户「${r.name || r.account}」吗？禁用后该用户将无法登录和调用模型。`,
      positiveText: '禁用',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await toggleUser(r.id)
          await load()
          message.success('已禁用')
        } catch (e) {
          message.error(e.msg || '操作失败')
        }
      }
    })
    return
  }
  try {
    await toggleUser(r.id)
    await load()
    message.success('已启用')
  } catch (e) {
    message.error(e.msg || '操作失败')
  }
}

function openReset(r) {
  resetTargetId.value = r.id
  resetPwd.value = ''
  showReset.value = true
}

async function doReset() {
  if (!resetPwd.value) {
    message.warning('密码不能为空')
    return
  }
  submitting.value = true
  try {
    await resetPasswordApi(resetTargetId.value, resetPwd.value)
    showReset.value = false
    message.success('已重置')
  } catch (e) {
    message.error(e.msg || '重置失败')
  } finally { submitting.value = false }
}

function openAssign(r) {
  assignTargetId.value = r.id
  assignRoleIds.value = r.role_ids ? [...r.role_ids] : []
  showAssign.value = true
}

async function doAssign() {
  submitting.value = true
  try {
    await assignRoles(assignTargetId.value, assignRoleIds.value)
    showAssign.value = false
    await load()
    message.success('已保存')
  } catch (e) {
    message.error(e.msg || '保存失败')
  } finally { submitting.value = false }
}

function goApiKeys(r) {
  router.push(`/admin/users/${r.id}/apikeys`)
}

function goRechargeRecords(r) {
  router.push(`/admin/users/${r.id}/recharge`)
}

function openRecharge(r) {
  rechargeTargetId.value = r.id
  rechargeAmount.value = 1
  rechargeRemark.value = ''
  showRecharge.value = true
}

async function doRecharge() {
  if (!rechargeAmount.value || rechargeAmount.value <= 0) {
    message.warning('金额必须大于 0')
    return
  }
  submitting.value = true
  try {
    await rechargeAdmin(rechargeTargetId.value, rechargeAmount.value, rechargeRemark.value)
    showRecharge.value = false
    await load()
    message.success('充值成功')
  } catch (e) {
    message.error(e.msg || '充值失败')
  } finally { submitting.value = false }
}

onMounted(() => {
  load()
  loadRoles()
})
</script>
