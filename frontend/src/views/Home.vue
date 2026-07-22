<template>
  <div class="layout">
    <div class="sidebar" :class="{ collapsed: collapsed }">
      <div class="brand">
        <span v-if="!collapsed">AI API</span>
        <svg v-else class="brand-icon-sm" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="26" height="26">
          <rect width="32" height="32" rx="6" fill="#333"/>
          <text x="16" y="22" text-anchor="middle" font-family="Arial,sans-serif" font-weight="bold" font-size="18" fill="#fff">AI</text>
        </svg>
      </div>
      <nav class="nav">
        <template v-for="menu in menus" :key="menu.id">
          <template v-if="menu.children && menu.children.length">
            <div class="nav-group">
              <div class="nav-group-title" @click="toggleGroup(menu.id)" :title="collapsed ? menu.name : undefined">
                <span>{{ collapsed ? '' : menu.name }}</span>
                <ChevronDownOutline v-if="!collapsed" class="arrow" :class="{ expanded: expandedGroups[menu.id] }" />
              </div>
              <div v-show="!collapsed && expandedGroups[menu.id]">
                <router-link
                  v-for="child in menu.children"
                  :key="child.id"
                  :to="child.path"
                  class="nav-item nav-sub"
                  :title="child.name"
                >
                  <component :is="getIcon(child.path)" class="nav-icon" v-if="getIcon(child.path)" />
                  <span>{{ child.name }}</span>
                </router-link>
              </div>
            </div>
          </template>
          <router-link
            v-else-if="menu.path"
            :to="menu.path"
            class="nav-item"
            :title="collapsed ? menu.name : undefined"
          >
            <component :is="getIcon(menu.path)" class="nav-icon" v-if="getIcon(menu.path)" />
            <span v-if="!collapsed">{{ menu.name }}</span>
          </router-link>
          <div v-else class="nav-group-title nav-group-disabled">
            <span>{{ collapsed ? '' : menu.name }}</span>
          </div>
        </template>
      </nav>
      <div class="sidebar-footer">
        <div class="user-row" v-if="!collapsed">
          <span class="user-name">{{ user?.name }}</span>
          <span class="user-balance" :style="{marginLeft:'auto'}">¥ {{ fix4(user?.budget) }}</span>
          <div class="logout-btn" @click="handleLogout" title="退出登录">
            <svg viewBox="0 0 512 512" width="16" height="16">
              <path d="M304 336v40a40 40 0 01-40 40H104a40 40 0 01-40-40V136a40 40 0 0140-40h152c22.09 0 48 17.91 48 40v40" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/>
              <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M368 336l80-80-80-80"/>
              <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M176 256h256"/>
            </svg>
          </div>
        </div>
        <div class="collapse-row">
          <span v-if="collapsed" class="collapsed-balance" :title="'余额 ¥ ' + fix4(user?.budget)">{{ fix4(user?.budget) }}</span>
          <div v-if="collapsed" class="sep-line"></div>
          <div class="collapse-btn" @click="toggleCollapse" :title="collapsed ? '展开菜单' : '折叠菜单'">
            <ChevronBackOutline v-if="!collapsed" />
            <ChevronForwardOutline v-else />
          </div>
        </div>
      </div>
    </div>
    <div class="main">
      <div class="content">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, markRaw } from 'vue'
import { useRouter } from 'vue-router'
import { useUser } from '../stores/user'
import { logout } from '../api'
import { fix4 } from '../utils'
import {
  AnalyticsOutline,
  BarChartOutline,
  KeyOutline,
  CubeOutline,
  CashOutline,
  PulseOutline,
  PeopleOutline,
  ServerOutline,
  GridOutline,
  ChevronDownOutline,
  ChevronBackOutline,
  ChevronForwardOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const { user, menus, fetchUser } = useUser()

const collapsed = ref(false)
const expandedGroups = reactive({})

const iconMap = {
  '/usage':           markRaw(BarChartOutline),
  '/apikeys':         markRaw(KeyOutline),
  '/models':          markRaw(CubeOutline),
  '/recharge':        markRaw(CashOutline),
  '/admin/dashboard': markRaw(PulseOutline),
  '/admin/users':     markRaw(PeopleOutline),
  '/admin/providers': markRaw(ServerOutline),
  '/admin/models':    markRaw(GridOutline),
  '/admin/usage':     markRaw(AnalyticsOutline),
  '/admin/recharge':  markRaw(CashOutline),
}

function getIcon(path) {
  return iconMap[path] || null
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

async function handleLogout() {
  try { await logout() } catch {}
  router.replace('/login')
}

function toggleGroup(id) {
  expandedGroups[id] = !expandedGroups[id]
}

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

if (!user.value) {
  fetchUser().then(() => {
    menus.value?.forEach(m => {
      if (m.children && m.children.length) {
        expandedGroups[m.id] = true
      }
    })
    const first = firstNavigable(menus.value)
    if (first && router.currentRoute.value.path === '/home') {
      router.replace(first)
    }
  }).catch(() => router.replace('/login'))
}
</script>

<style>
:root {
  --bg-page: #f9f8f7;
  --border-color: #e0e0e0;
}
.layout { display: flex; height: 100vh; margin: 0; }

/* ===== 侧栏 ===== */
.sidebar {
  width: 220px;
  background: var(--bg-page);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow-y: auto;
  border-right: 1px solid var(--border-color);
  transition: width 0.2s ease;
}
.sidebar.collapsed { width: 60px; }

.brand {
  color: #333;
  font-size: 18px;
  font-weight: 700;
  text-align: center;
  padding: 20px 0 18px;
  letter-spacing: 3px;
  border-bottom: 1px solid var(--border-color);
  overflow: hidden;
  white-space: nowrap;
}
.brand-icon-sm { margin: 0 auto; display: block; }

.nav { flex: 1; padding: 8px 0; }

/* 单菜单项 */
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 20px;
  color: #555;
  text-decoration: none;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.15s;
  border-left: 3px solid transparent;
  margin: 1px 0;
  white-space: nowrap;
  overflow: hidden;
}
.collapsed .nav-item {
  padding: 10px 0;
  justify-content: center;
  border-left: none;
}
.nav-item:hover {
  color: #18a058;
  background: #e8f5e9;
}
.nav-item.router-link-active {
  color: #18a058;
  background: #e8f5e9;
  border-left-color: #18a058;
  font-weight: 600;
}
.collapsed .nav-item.router-link-active {
  border-left: none;
  position: relative;
}
.collapsed .nav-item.router-link-active::after {
  content: '';
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 3px;
  background: #18a058;
  border-radius: 0 2px 2px 0;
}
.nav-icon { width: 18px; height: 18px; flex-shrink: 0; }

.nav-sub { padding-left: 48px; font-size: 13px; }
.collapsed .nav-sub { padding-left: 0; }
.nav-sub .nav-icon { width: 16px; height: 16px; }

/* 分组 */
.nav-group-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  color: #999;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 1px;
  cursor: pointer;
  user-select: none;
  transition: color 0.15s;
  white-space: nowrap;
  overflow: hidden;
}
.collapsed .nav-group-title {
  padding: 10px 0;
  justify-content: center;
}
.nav-group-title:hover { color: #666; }
.nav-group-disabled { cursor: default; }
.nav-group-disabled:hover { color: #999; }

.arrow { transition: transform 0.2s; opacity: 0.5; display: flex; width: 14px; height: 14px; }
.arrow.expanded { transform: rotate(180deg); }

/* 底部：用户信息 + 操作 */
.sidebar-footer {
  border-top: 1px solid var(--border-color);
}
.collapsed .sidebar-footer { padding: 8px; }

.user-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  font-size: 13px;
}
.user-name { font-weight: 600; color: #333; white-space: nowrap; }
.user-balance { font-weight: 700; color: #e53935; white-space: nowrap; }

.collapse-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 6px 0 8px;
  border-top: 1px solid var(--border-color);
}
.collapsed .collapse-row { border-top: none; flex-direction: column; gap: 6px; }
.sep-line { width: 20px; height: 1px; background: var(--border-color); }
.collapsed-balance { font-size: 10px; font-weight: 700; color: #e53935; }

.collapse-btn, .logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  color: #333;
  cursor: pointer;
  transition: color 0.15s;
  border-radius: 4px;
}
.collapse-btn:hover { color: #18a058; background: #e8f5e9; }
.logout-btn:hover { color: #e53935; background: #fce4ec; }

.collapse-btn svg, .logout-btn svg { width: 16px; height: 16px; }

/* ===== 主区域 ===== */
.main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.content { flex: 1; padding: 20px; background: var(--bg-page); overflow-y: auto; }
</style>
