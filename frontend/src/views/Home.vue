<template>
  <div class="layout">
    <div class="sidebar">
      <div class="brand">AI API</div>
      <template v-for="menu in menus" :key="menu.id">
        <!-- 有子菜单：作为分组容器，点击展开/收起 -->
        <template v-if="menu.children && menu.children.length">
          <div class="menu-group-title" @click="toggleGroup(menu.id)">
            <span>{{ menu.name }}</span>
            <span class="arrow" :class="{ expanded: expandedGroups[menu.id] }">▾</span>
          </div>
          <router-link
            v-show="expandedGroups[menu.id]"
            v-for="child in menu.children"
            :key="child.id"
            :to="child.path"
            class="menu-item menu-sub"
          >{{ child.name }}</router-link>
        </template>
        <!-- 无子菜单且 path 非空：直接跳转 -->
        <router-link
          v-else-if="menu.path"
          :to="menu.path"
          class="menu-item"
        >{{ menu.name }}</router-link>
        <!-- 无子菜单且 path 为空：纯分组标题，不可点击 -->
        <div v-else class="menu-group-title menu-group-disabled">
          <span>{{ menu.name }}</span>
        </div>
      </template>
    </div>
    <div class="main">
      <div class="topbar">
        <div class="topbar-right">
          <span class="balance">¥ {{ fix4(user?.budget) }}</span>
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
import { reactive } from 'vue'
import { NTag, NButton } from 'naive-ui'
import { useRouter } from 'vue-router'
import { useUser } from '../stores/user'
import { logout } from '../api'
import { fix4 } from '../utils'

const router = useRouter()
const { user, menus, fetchUser } = useUser()

// 展开的分组 id 集合（默认全部展开）
const expandedGroups = reactive({})

async function handleLogout() {
  try { await logout() } catch {}
  router.replace('/login')
}

function toggleGroup(id) {
  expandedGroups[id] = !expandedGroups[id]
}

if (!user.value) {
  fetchUser().then(() => {
    // 初始化：默认展开所有有子菜单的分组
    menus.value?.forEach(m => {
      if (m.children && m.children.length) {
        expandedGroups[m.id] = true
      }
    })
    // 跳转到第一个可跳转的菜单（有 path 且无子菜单，或第一个有子菜单的子项）
    const first = firstNavigable(menus.value)
    if (first && router.currentRoute.value.path === '/home') {
      router.replace(first)
    }
  }).catch(() => router.replace('/login'))
}

// 找第一个可导航的 path（扁平或子菜单中第一个有 path 的）
function firstNavigable(list) {
  for (const m of list || []) {
    if (m.children?.length) {
      const sub = firstNavigable(m.children)
      if (sub) return sub
    } else if (m.path) {
      return m.path
    }
  }
  return null
}
</script>

<style>
.layout { display: flex; height: 100vh; margin: 0; }
.sidebar { width: 200px; background: #001529; display: flex; flex-direction: column; flex-shrink: 0; overflow-y: auto; }
.brand { color: #fff; font-size: 18px; font-weight: 700; text-align: center; padding: 20px 0; letter-spacing: 3px; }
.menu-item { display: block; padding: 14px 20px; color: #fff9; text-decoration: none; font-size: 14px; cursor: pointer; }
.menu-sub { padding-left: 36px; font-size: 13px; }
.menu-item:hover, .menu-item.router-link-active { background: #1677ff; color: #fff; }
.menu-group-title { display: flex; align-items: center; justify-content: space-between; padding: 14px 20px; color: #fffc; font-size: 14px; cursor: pointer; user-select: none; }
.menu-group-title:hover { background: #00203a; }
.menu-group-disabled { cursor: default; color: #fff6; }
.menu-group-disabled:hover { background: transparent; }
.arrow { font-size: 10px; transition: transform 0.2s; opacity: 0.6; }
.arrow.expanded { transform: rotate(180deg); }
.main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.topbar { display: flex; align-items: center; justify-content: flex-end; height: 56px; padding: 0 24px; background: #fff; border-bottom: 1px solid #eee; flex-shrink: 0; }
.topbar-right { display: flex; align-items: center; gap: 12px; }
.balance { font-size: 15px; font-weight: 600; color: #18a058; }
.content { flex: 1; padding: 20px; background: #f5f7fa; overflow-y: auto; }
</style>
