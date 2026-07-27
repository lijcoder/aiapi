import { createRouter, createWebHistory } from 'vue-router'
import { tryRestoreSession, getAccessToken } from '../api'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import Recharge from '../views/Recharge.vue'
import Models from '../views/Models.vue'
import ApiKeys from '../views/ApiKeys.vue'
import Usage from '../views/Usage.vue'
import Profile from '../views/Profile.vue'
import ChangePassword from '../views/ChangePassword.vue'

// Admin pages
import AdminDashboard from '../views/admin/Dashboard.vue'
import AdminUsers from '../views/admin/Users.vue'
import AdminUserApiKeys from '../views/admin/UserApiKeys.vue'
import AdminProviders from '../views/admin/Providers.vue'
import AdminModels from '../views/admin/Models.vue'
import AdminUsage from '../views/admin/Usage.vue'
import AdminRecharge from '../views/admin/Recharge.vue'
import AdminLogs from '../views/admin/Logs.vue'

const routes = [
  { path: '/login', name: 'Login', component: Login },
  {
    path: '/',
    component: Home,
    redirect: '/home',
    children: [
      // 用户自助页面
      { path: 'usage', name: 'Usage', component: Usage },
      { path: 'apikeys', name: 'ApiKeys', component: ApiKeys },
      { path: 'models', name: 'Models', component: Models },
      { path: 'recharge', name: 'Recharge', component: Recharge },
      { path: 'profile', name: 'Profile', component: Profile },
      { path: 'profile/password', name: 'ChangePassword', component: ChangePassword },
      // 管理后台页面
      { path: 'admin/dashboard', name: 'AdminDashboard', component: AdminDashboard },
      { path: 'admin/users', name: 'AdminUsers', component: AdminUsers },
      { path: 'admin/users/:id/apikeys', name: 'AdminUserApiKeys', component: AdminUserApiKeys, props: true },
      { path: 'admin/users/:id/recharge', name: 'AdminUserRecharge', component: AdminRecharge, props: true },
      { path: 'admin/providers', name: 'AdminProviders', component: AdminProviders },
      { path: 'admin/models', name: 'AdminModels', component: AdminModels },
      { path: 'admin/usage', name: 'AdminUsage', component: AdminUsage },
      { path: 'admin/recharge', name: 'AdminRecharge', component: AdminRecharge },
      { path: 'admin/logs', name: 'AdminLogs', component: AdminLogs },
      // 兜底：/home 不渲染内容，仅触发 Home.vue 的菜单跳转逻辑
      { path: 'home', name: 'HomePlaceholder', component: { render: () => null } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 全局前置守卫：页面刷新后 access token 丢失，靠 refresh cookie 静默恢复
// /login 探测 refresh cookie：有效则跳首页（已登录），无效则显示登录页
router.beforeEach(async (to) => {
  if (to.path === '/login') {
    const ok = await tryRestoreSession()
    return ok ? '/' : true
  }
  // 其它页面：确保有 access token，没有就尝试恢复
  if (!getAccessToken()) {
    const ok = await tryRestoreSession()
    if (!ok) return '/login'
  }
  return true
})

export default router
