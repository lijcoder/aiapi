<template>
  <div class="login-page">
    <n-card class="login-card" :bordered="false" size="large">
      <template #header>
        <div class="card-title">AI API 管理后台</div>
      </template>

      <!-- 第一步：账号密码 -->
      <template v-if="!pendingTicket">
        <n-input v-model:value="account" placeholder="账号" size="large" />
        <n-input v-model:value="password" type="password" placeholder="密码" show-password-on="click" size="large" style="margin-top:16px" @keydown.enter="handleLogin" />
        <p v-if="error" class="error">{{ error }}</p>
        <n-button type="primary" size="large" :loading="loading" block @click="handleLogin" style="margin-top:20px">登录</n-button>
      </template>

      <!-- 第二步：TOTP 验证码（账号密码已通过，用户开启了 2FA） -->
      <template v-else>
        <p class="tip">该账号已开启两步验证，请输入 Authenticator 中的 6 位验证码</p>
        <n-input v-model:value="code" placeholder="6 位验证码" size="large" maxlength="6" :allow-input="onlyDigits" @keydown.enter="handleLogin2fa" />
        <p v-if="error" class="error">{{ error }}</p>
        <n-button type="primary" size="large" :loading="loading" block @click="handleLogin2fa" style="margin-top:20px">验证并登录</n-button>
        <n-button text size="small" block style="margin-top:10px" @click="backToPassword">返回重新登录</n-button>
      </template>
    </n-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { NCard, NInput, NButton } from 'naive-ui'
import { useRouter } from 'vue-router'
import { login, login2fa } from '../api'
import { useUser } from '../stores/user'

const router = useRouter()
const { fetchUser } = useUser()
const account = ref('')
const password = ref('')
const code = ref('')
const pendingTicket = ref('')
const loading = ref(false)
const error = ref('')

const onlyDigits = v => /^\d*$/.test(v)

async function enterApp() {
  const { menus } = useUser()
  await fetchUser()
  const first = menus.value?.[0]
  router.replace(first ? first.path : '/')
}

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    const data = await login(account.value, password.value)
    if (data.need_2fa) {
      pendingTicket.value = data.pending_ticket
      code.value = ''
      return
    }
    await enterApp()
  } catch (e) {
    error.value = e.msg || '登录失败'
  } finally { loading.value = false }
}

async function handleLogin2fa() {
  error.value = ''
  if (!/^\d{6}$/.test(code.value)) {
    error.value = '请输入 6 位数字验证码'
    return
  }
  loading.value = true
  try {
    await login2fa(pendingTicket.value, code.value)
    await enterApp()
  } catch (e) {
    error.value = e.msg || '验证失败'
    // 票据过期/作废 → 回到账号密码步骤
    if (e.status === 401) backToPassword()
  } finally { loading.value = false }
}

function backToPassword() {
  pendingTicket.value = ''
  code.value = ''
  password.value = ''
  error.value = ''
}
</script>

<style>
.login-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; background: #f9f8f7; }
.login-card { width: 400px; border-radius: 12px !important; box-shadow: 0 2px 16px rgba(0,0,0,0.08); }
.login-card .n-card-header { text-align: center; }
.card-title { font-size: 20px; font-weight: 600; color: #333; }
.error { color: #d03050; font-size: 13px; text-align: center; margin: 12px 0 0; }
.tip { color: #666; font-size: 13px; margin: 0 0 14px; }
</style>
