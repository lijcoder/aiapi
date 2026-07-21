<template>
  <div class="layout">
    <div class="sidebar">
      <div class="brand">AI API</div>
      <router-link to="/usage" class="menu-item">使用统计</router-link>
      <router-link to="/apikeys" class="menu-item">API 密钥</router-link>
      <router-link to="/models" class="menu-item">模型列表</router-link>
      <router-link to="/recharge" class="menu-item">充值中心</router-link>
    </div>
    <div class="main">
      <div class="topbar">
        <div class="topbar-right">
          <span class="balance">¥ {{ user?.budget?.toFixed(2) }}</span>
          <n-tag size="small">{{ user?.name }}</n-tag>
          <n-button text type="error" size="small" @click="handleLogout">退出</n-button>
        </div>
      </div>
      <div class="content">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script setup>
import { NTag, NButton } from 'naive-ui'
import { useRouter } from 'vue-router'
import { useUser } from '../stores/user'
import { logout } from '../api'

const router = useRouter()
const { user, fetchUser } = useUser()

async function handleLogout() {
  try { await logout() } catch {}
  router.replace('/login')
}

if (!user.value) fetchUser().catch(() => router.replace('/login'))
</script>

<style>
.layout { display: flex; height: 100vh; margin: 0; }
.sidebar { width: 200px; background: #001529; display: flex; flex-direction: column; flex-shrink: 0; }
.brand { color: #fff; font-size: 18px; font-weight: 700; text-align: center; padding: 20px 0; letter-spacing: 3px; }
.menu-item { display: block; padding: 14px 20px; color: #fff9; text-decoration: none; font-size: 14px; }
.menu-item:hover, .menu-item.router-link-active { background: #1677ff; color: #fff; }
.main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.topbar { display: flex; align-items: center; justify-content: flex-end; height: 56px; padding: 0 24px; background: #fff; border-bottom: 1px solid #eee; flex-shrink: 0; }
.topbar-right { display: flex; align-items: center; gap: 12px; }
.balance { font-size: 15px; font-weight: 600; color: #18a058; }
.content { flex: 1; padding: 20px; background: #f5f7fa; overflow-y: auto; }
</style>
