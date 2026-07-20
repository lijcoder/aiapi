<template>
  <div class="login-page">
    <form class="login-form" @submit.prevent="handleLogin">
      <h2>AI API 管理后台</h2>
      <input v-model="account" placeholder="账号" autocomplete="username" />
      <input v-model="password" type="password" placeholder="密码" autocomplete="current-password" />
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="loading">{{ loading ? '登录中...' : '登录' }}</button>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login as loginApi } from '../api'
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
    await loginApi(account.value, password.value)
    await fetchUser()
    router.replace('/')
  } catch (e) {
    error.value = e.msg || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex; align-items: center; justify-content: center;
  min-height: 100vh; background: #f5f5f5;
}
.login-form {
  background: #fff; padding: 40px; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,.1);
  width: 320px; display: flex; flex-direction: column; gap: 16px;
}
.login-form h2 { text-align: center; margin: 0 0 8px; color: #333; }
.login-form input {
  padding: 10px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;
}
.login-form button {
  padding: 10px; background: #1a73e8; color: #fff; border: none;
  border-radius: 4px; font-size: 14px; cursor: pointer;
}
.login-form button:disabled { opacity: .6; }
.error { color: #d32f2f; font-size: 13px; text-align: center; margin: 0; }
</style>
