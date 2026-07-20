<template>
  <div class="recharge-page">
    <div class="card">
      <div class="card-header">
        <h3>充值流水</h3>
        <button class="btn-primary" @click="showModal = true">充值</button>
      </div>
      <table v-if="records.length">
        <thead>
          <tr><th>ID</th><th>金额</th><th>充值前</th><th>充值后</th><th>操作人</th><th>备注</th><th>时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="r in records" :key="r.id">
            <td>{{ r.id }}</td>
            <td class="val-plus">¥ {{ r.amount }}</td>
            <td>¥ {{ r.balance_before }}</td>
            <td>¥ {{ r.balance_after }}</td>
            <td>{{ r.operator }}</td>
            <td>{{ r.remark }}</td>
            <td>{{ formatTime(r.created_at) }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">暂无充值记录</p>
    </div>

    <!-- 充值弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <h3>账户充值</h3>
        <input v-model.number="amount" type="number" step="0.01" min="0.01" placeholder="充值金额" />
        <input v-model="remark" placeholder="备注（选填）" />
        <p v-if="msg" :class="ok ? 'msg-ok' : 'msg-err'">{{ msg }}</p>
        <div class="modal-btns">
          <button class="btn-cancel" @click="closeModal">取消</button>
          <button class="btn-primary" @click="doRecharge" :disabled="loading">
            {{ loading ? '充值中...' : '确认充值' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useUser } from '../stores/user'
import { rechargeSelf, rechargeSelfRecords } from '../api'

const { fetchUser } = useUser()

// 弹窗
const showModal = ref(false)
const amount = ref(0)
const remark = ref('')
const loading = ref(false)
const msg = ref('')
const ok = ref(true)

function closeModal() {
  showModal.value = false
  amount.value = 0
  remark.value = ''
  msg.value = ''
}

async function doRecharge() {
  msg.value = ''
  if (amount.value <= 0) return
  loading.value = true
  try {
    await rechargeSelf(amount.value, remark.value)
    await fetchUser()
    await loadRecords()
    closeModal()
  } catch (e) {
    msg.value = e.msg || '充值失败'
    ok.value = false
  } finally { loading.value = false }
}

// 流水
const records = ref([])

async function loadRecords() {
  try {
    records.value = await rechargeSelfRecords()
  } catch {}
}

function formatTime(t) {
  if (!t || t.startsWith('0001')) return '-'
  return t.replace('T', ' ').substring(0, 19)
}

onMounted(() => loadRecords())
</script>

<style scoped>
.recharge-page { display: flex; flex-direction: column; gap: 20px; }
.card { background: #fff; border-radius: 6px; padding: 24px; box-shadow: 0 1px 3px rgba(0,0,0,.05); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.card-header h3 { margin: 0; font-size: 15px; font-weight: 600; color: #1e293b; }

.btn-primary { padding: 8px 20px; background: #3b82f6; color: #fff; border: none; border-radius: 4px; font-size: 14px; cursor: pointer; white-space: nowrap; }
.btn-primary:hover { background: #2563eb; }
.btn-primary:disabled { opacity: .5; cursor: not-allowed; }
.btn-cancel { padding: 8px 20px; background: #f1f5f9; color: #475569; border: 1px solid #e2e8f0; border-radius: 4px; font-size: 14px; cursor: pointer; }

table { width: 100%; border-collapse: collapse; font-size: 13px; }
th, td { padding: 10px 8px; border-bottom: 1px solid #f1f5f9; text-align: left; }
th { color: #94a3b8; font-weight: 500; font-size: 12px; }
td { color: #334155; }
.val-plus { color: #16a34a; font-weight: 500; }
.empty { color: #94a3b8; font-size: 13px; }

/* 弹窗 */
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.3); display: flex;
  align-items: center; justify-content: center; z-index: 100;
}
.modal {
  background: #fff; border-radius: 8px; padding: 28px; width: 380px;
  box-shadow: 0 8px 30px rgba(0,0,0,.12); display: flex; flex-direction: column; gap: 14px;
}
.modal h3 { margin: 0; font-size: 16px; font-weight: 600; color: #1e293b; }
.modal input { padding: 9px 10px; border: 1px solid #e2e8f0; border-radius: 4px; font-size: 14px; outline: none; }
.modal input:focus { border-color: #3b82f6; }
.modal-btns { display: flex; justify-content: flex-end; gap: 10px; margin-top: 4px; }
.msg-ok { color: #16a34a; font-size: 13px; margin: 0; }
.msg-err { color: #dc2626; font-size: 13px; margin: 0; }
</style>
