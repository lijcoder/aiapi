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

export function rechargeRecords(userId) {
  if (userId) return request('/recharge/records', { userId })
  return request('/recharge/records/self')
}
