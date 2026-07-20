<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="logo">AI API</div>
      <nav>
        <router-link to="/" class="nav-item">充值</router-link>
      </nav>
    </aside>
    <div class="main">
      <header class="topbar">
        <div class="topbar-left"></div>
        <div class="topbar-right">
          <span class="user-budget">余额 ¥ {{ user?.budget?.toFixed(2) }}</span>
          <span class="user-name">{{ user?.name }}</span>
          <button class="btn-logout" @click="handleLogout">退出</button>
        </div>
      </header>
      <div class="content">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useUser } from '../stores/user'
import { logout } from '../api'

const router = useRouter()
const { user, fetchUser } = useUser()

async function handleLogout() {
  try { await logout() } catch {}
  router.replace('/login')
}

if (!user.value) {
  fetchUser().catch(() => router.replace('/login'))
}
</script>

<style scoped>
.layout { display: flex; min-height: 100vh; }
.sidebar {
  width: 180px; background: #1e293b; color: #cbd5e1; display: flex;
  flex-direction: column; flex-shrink: 0;
}
.logo { padding: 20px 16px; font-size: 18px; font-weight: 700; color: #fff; letter-spacing: 2px; }
.nav-item {
  display: block; padding: 12px 16px; color: #94a3b8; text-decoration: none;
  font-size: 14px; transition: all .15s;
}
.nav-item:hover, .nav-item.router-link-active { background: #334155; color: #fff; }
.main { flex: 1; display: flex; flex-direction: column; background: #f1f5f9; }
.topbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 24px; height: 52px; background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,.06);
  flex-shrink: 0;
}
.topbar-right { display: flex; align-items: center; gap: 16px; }
.user-name { font-size: 14px; font-weight: 500; color: #334155; }
.user-budget { font-size: 13px; color: #16a34a; font-weight: 600; }
.btn-logout { padding: 5px 14px; border: 1px solid #dc2626; color: #dc2626; background: none; border-radius: 4px; font-size: 13px; cursor: pointer; }
.content { flex: 1; padding: 20px 24px; overflow-y: auto; }
</style>
