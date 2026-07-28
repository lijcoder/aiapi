const BASE = '/manager'

// ===== access token 内存存储 + 自动 refresh =====
// access JWT 仅存内存，刷新页面会丢失，靠 silent refresh（带 refresh cookie）恢复
let accessToken = ''
let refreshPromise = null // 并发请求合并：多个请求同时 401 时只触发一次 /refresh

// 业务码：access 过期，前端触发 /refresh 续期
const CODE_TOKEN_EXPIRED = 1016

function isAuthPath(path) {
  return path === '/login' || path === '/login/2fa' || path === '/refresh'
}

// 静默刷新：带 refresh cookie，不带 access
function doRefresh() {
  if (refreshPromise) return refreshPromise
  refreshPromise = fetch(`${BASE}/refresh`, {
    method: 'POST',
    credentials: 'include'
  })
    .then(r => r.json())
    .then(d => {
      if (d.code) throw d
      accessToken = d.data.access_token
      return accessToken
    })
    .finally(() => { refreshPromise = null })
  return refreshPromise
}

async function request(path, body) {
  const headers = { 'Content-Type': 'application/json' }
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  const resp = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers,
    body: body ? JSON.stringify(body) : '{}'
  })

  // 401 且非鉴权接口本身 → 尝试 refresh 后重试一次
  if (resp.status === 401 && !isAuthPath(path)) {
    const data = await resp.json().catch(() => ({}))
    if (data.code === CODE_TOKEN_EXPIRED) {
      try {
        await doRefresh()
        return request(path, body) // 重试原请求
      } catch {
        accessToken = ''
        throw { code: data.code, msg: '登录已过期，请重新登录', status: 401 }
      }
    }
    accessToken = ''
    throw { code: data.code, msg: data.msg || '未登录', status: 401 }
  }

  const data = await resp.json()
  if (data.code) {
    throw { code: data.code, msg: data.msg, status: resp.status }
  }
  return data.data
}

// ===== 鉴权接口 =====

// 登录第一步：账号密码。若用户已开启 2FA，返回 { need_2fa: true, pending_ticket }，
// 不设置 accessToken，前端进入验证码步骤后调 login2fa。
export function login(account, password) {
  return request('/login', { account, password }).then(data => {
    if (data.need_2fa) return data
    accessToken = data.access_token
    return data
  })
}

// 登录第二步：pending 票据 + TOTP 验证码，通过则拿到 token
export function login2fa(pendingTicket, code) {
  return request('/login/2fa', { pending_ticket: pendingTicket, code }).then(data => {
    accessToken = data.access_token
    return data
  })
}

export function logout() {
  return request('/logout', {}).finally(() => {
    accessToken = ''
  })
}

// 页面刷新后恢复会话：尝试 silent refresh，成功返回 true
export async function tryRestoreSession() {
  try {
    await doRefresh()
    return true
  } catch {
    return false
  }
}

// 读取当前 access token（路由守卫用，判断内存是否已有）
export function getAccessToken() {
  return accessToken
}

// 清除内存中的 access token（改密/登出后调用，后端 session 已被吊销）
export function clearAccessToken() {
  accessToken = ''
}

export function self() {
  return request('/self')
}

export function rechargeSelf(amount, remark) {
  return request('/recharge/self', { amount, remark })
}

export function rechargeAdmin(userId, amount, remark) {
  return request('/recharge', { userId, amount, remark })
}

export function rechargeSelfRecords(page = 1, pageSize = 20) {
  return request('/recharge/records/self', { page, page_size: pageSize })
}

export function listModels(provider = '', model = '', page = 1, pageSize = 20) {
  return request('/models', { provider, model, page, page_size: pageSize })
}

export function listMyApiKeys(page = 1, pageSize = 20) {
  return request('/apikeys/list/self', { page, page_size: pageSize })
}

export function createApiKey(name, budget, unlimited) {
  return request('/apikeys/create/self', { name, budget, unlimited })
}

export function toggleApiKey(id) {
  return request('/apikeys/toggle/self', { id })
}

export function deleteApiKey(id) {
  return request('/apikeys/delete/self', { id })
}

export function renameApiKey(id, name) {
  return request('/apikeys/rename/self', { id, name })
}

export function updateApiKeyBudget(id, budget, unlimited) {
  return request('/apikeys/budget/self', { id, budget, unlimited })
}

export function rechargeRecords(userId, page = 1, pageSize = 20) {
  if (userId) return request('/recharge/records', { userId, page, page_size: pageSize })
  return request('/recharge/records/self', { page, page_size: pageSize })
}

export function usageStats(mode, startDate, endDate, apiKeyId, model, provider, groupBy) {
  return request('/usage/stats/self', { mode, start_date: startDate, end_date: endDate, api_key_id: apiKeyId || 0, model: model || '', provider: provider || '', group_by: groupBy || '' })
}

export function usageFilters() {
  return request('/usage/filters/self')
}

// ===== 个人设置 =====

export function updateProfileSelf(name, email) {
  return request('/profile/update/self', { name, email })
}

export function updatePasswordSelf(oldPassword, newPassword) {
  return request('/profile/password/self', { old_password: oldPassword, new_password: newPassword })
}

// ===== 两步验证（2FA）=====

// 生成 TOTP 密钥与二维码（未生效，确认后落库）
export function setup2faSelf() {
  return request('/2fa/setup/self')
}

// 校验首个验证码，确认绑定
export function confirm2faSelf(setupTicket, code) {
  return request('/2fa/confirm/self', { setup_ticket: setupTicket, code })
}

// 校验密码后关闭 2FA
export function disable2faSelf(password) {
  return request('/2fa/disable/self', { password })
}

// ===== 超管：用户管理 =====

export function listUsers(keyword, page = 1, pageSize = 20) {
  return request('/users/list', { keyword: keyword || '', page, page_size: pageSize })
}

export function getUser(id) {
  return request('/users/get', { id })
}

export function createUser(name, account, password, budget, unlimited) {
  return request('/users/create', { name, account, password, budget, unlimited })
}

export function updateUser(id, name, budget, unlimited) {
  return request('/users/update', { id, name, budget, unlimited })
}

export function toggleUser(id) {
  return request('/users/toggle', { id })
}

export function resetPassword(id, password) {
  return request('/users/reset-password', { id, password })
}

export function assignRoles(id, roleIds) {
  return request('/users/assign-roles', { id, role_ids: roleIds })
}

export function listRoles() {
  return request('/roles/list')
}

// ===== 超管：Provider 管理 =====

export function listProviders(page = 1, pageSize = 20) {
  return request('/providers/list', { page, page_size: pageSize })
}

export function createProvider(type, domain, headers) {
  return request('/providers/create', { type, domain, headers })
}

export function updateProvider(type, domain, headers) {
  return request('/providers/update', { type, domain, headers })
}

export function toggleProvider(type) {
  return request('/providers/toggle', { type })
}

// ===== 超管：模型管理 =====

export function listModelsAdmin(provider, model, page = 1, pageSize = 20) {
  return request('/models/list', { provider: provider || '', model: model || '', page, page_size: pageSize })
}

export function createModel(data) {
  return request('/models/create', data)
}

export function updateModel(data) {
  return request('/models/update', data)
}

export function deleteModel(id) {
  return request('/models/delete', { id })
}

// ===== 超管：管理指定用户的 API Key =====

export function listUserApiKeys(userId, page = 1, pageSize = 20) {
  return request('/apikeys/list', { user_id: userId, page, page_size: pageSize })
}

export function toggleUserApiKey(id) {
  return request('/apikeys/toggle', { id })
}

export function deleteUserApiKey(id) {
  return request('/apikeys/delete', { id })
}

export function renameUserApiKey(id, name) {
  return request('/apikeys/rename', { id, name })
}

export function updateUserApiKeyBudget(id, budget, unlimited) {
  return request('/apikeys/budget', { id, budget, unlimited })
}

// ===== API Key 模型访问策略 =====

export function getApiKeyModelAccess(apiKeyId) {
  return request('/apikeys/models/get', { api_key_id: apiKeyId })
}

export function setApiKeyModelAccess(apiKeyId, modelPolicy, modelIds) {
  return request('/apikeys/models/set', { api_key_id: apiKeyId, model_policy: modelPolicy, model_ids: modelIds })
}

export function getApiKeyModelAccessSelf(apiKeyId) {
  return request('/apikeys/models/get/self', { api_key_id: apiKeyId })
}

export function setApiKeyModelAccessSelf(apiKeyId, modelPolicy, modelIds) {
  return request('/apikeys/models/set/self', { api_key_id: apiKeyId, model_policy: modelPolicy, model_ids: modelIds })
}

// ===== 超管：全平台充值流水 =====

export function listAllRechargeRecords(keyword, page = 1, pageSize = 20) {
  return request('/recharge/records/list', { keyword: keyword || '', page, page_size: pageSize })
}

// ===== 超管：全局统计 =====

export function usageStatsAdmin(mode, startDate, endDate, userId, apiKeyId, model, provider, groupBy) {
  return request('/usage/stats', {
    mode, start_date: startDate, end_date: endDate,
    user_id: userId || 0, api_key_id: apiKeyId || 0,
    model: model || '', provider: provider || '', group_by: groupBy || ''
  })
}

export function usageFiltersAdmin() {
  return request('/usage/filters')
}

// ===== 超管：日志查询 =====

export function fetchLogs() {
  return request('/logs')
}

// ===== 超管：仪表盘 =====

export function dashboardOverview() {
  return request('/dashboard')
}
