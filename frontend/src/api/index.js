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
