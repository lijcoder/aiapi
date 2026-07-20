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
