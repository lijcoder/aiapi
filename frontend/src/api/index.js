const BASE = '/manager'

async function request(path, body) {
  const resp = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : '{}'
  })
  const data = await resp.json()
  if (data.code) {
    throw { code: data.code, msg: data.msg, status: resp.status }
  }
  return data.data
}

export function login(account, password) {
  return request('/login', { account, password })
}

export function logout() {
  return request('/logout')
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

export function rechargeSelfRecords() {
  return request('/recharge/records/self')
}

export function listModels() {
  return request('/models')
}

export function listMyApiKeys() {
  return request('/apikeys/list/self')
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

export function rechargeRecords(userId) {
  if (userId) return request('/recharge/records', { userId })
  return request('/recharge/records/self')
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

// ===== 超管：用户管理 =====

export function listUsers(keyword) {
  return request('/users/list', { keyword: keyword || '' })
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

export function listProviders() {
  return request('/providers/list')
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

export function listModelsAdmin(provider, model) {
  return request('/models/list', { provider: provider || '', model: model || '' })
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

export function listUserApiKeys(userId) {
  return request('/apikeys/list', { user_id: userId })
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

export function listAllRechargeRecords(keyword) {
  return request('/recharge/records/list', { keyword: keyword || '' })
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
