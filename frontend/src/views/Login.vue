<template>
  <div class="login-page">
    <n-card class="login-card" :bordered="false" size="large">
      <template #header>
        <div class="card-title">AI API 管理后台</div>
      </template>
      <n-input v-model:value="account" placeholder="账号" size="large" />
      <n-input v-model:value="password" type="password" placeholder="密码" show-password-on="click" size="large" style="margin-top:16px" @keydown.enter="handleLogin" />
      <p v-if="error" class="error">{{ error }}</p>
      <n-button type="primary" size="large" :loading="loading" block @click="handleLogin" style="margin-top:20px">登录</n-button>
    </n-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { NCard, NInput, NButton } from 'naive-ui'
import { useRouter } from 'vue-router'
import { login } from '../api'
import { useUser } from '../stores/user'

const router = useRouter()
const { fetchUser } = useUser()
const account = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await login(account.value, password.value)
    const { menus } = useUser()
    await fetchUser()
    const first = menus.value?.[0]
    router.replace(first ? first.path : '/')
  } catch (e) {
    error.value = e.msg || '登录失败'
  } finally { loading.value = false }
}
</script>

<style>
.login-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; background: linear-gradient(135deg, #f5f5f5 0%, #e8e8e8 50%, #d6d6d6 100%); }
.login-card { width: 400px; border-radius: 12px !important; box-shadow: 0 2px 16px rgba(0,0,0,0.08); }
.login-card .n-card-header { text-align: center; }
.card-title { font-size: 20px; font-weight: 600; color: #333; }
.error { color: #d03050; font-size: 13px; text-align: center; margin: 12px 0 0; }
</style>
