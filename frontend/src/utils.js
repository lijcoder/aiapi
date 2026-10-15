// fix4 格式化金额：最多4位小数，去掉尾部无意义的 0
// 0.0000 → "0", 0.1000 → "0.1", 0.1234 → "0.1234"
export function fix4(n) {
  if (n == null) return '-'
  return Number(n).toFixed(4).replace(/\.?0+$/, '')
}

// formatTime 格式化时间字符串
export function formatTime(t) {
  if (!t || t.startsWith('0001')) return '-'
  return t.replace('T', ' ').substring(0, 19)
}

// copyText 复制文本到剪贴板，返回是否成功。
// navigator.clipboard 只在安全上下文（HTTPS 或 localhost）可用：用 http://<ip>:<port>
// 访问管理台时它是 undefined，直接调用会抛异常，所以回退到 execCommand 方案。
// 约定：本函数不抛异常，失败返回 false，由调用方按返回值提示。
export async function copyText(text) {
  const value = text == null ? '' : String(text)
  if (!value) return false
  if (window.isSecureContext && navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch (e) {
      // 权限被拒等场景继续走回退方案；只记失败原因，不记内容（内容含 API Key 明文）
      console.warn('clipboard.writeText 失败，回退 execCommand', e)
    }
  }
  return legacyCopy(value)
}

// legacyCopy 通过临时 textarea + document.execCommand('copy') 复制，非安全上下文下仍可用。
// 注意 execCommand 依赖用户手势，且 Safari 的手势令牌只在当前任务内有效——
// 它必须与点击在同一任务里同步执行，跨 await 调用会失败（见 ApiKeys.vue 的兜底弹窗）。
function legacyCopy(value) {
  let ta
  try {
    ta = document.createElement('textarea')
    ta.value = value
    ta.setAttribute('readonly', '')
    ta.style.cssText = 'position:fixed;top:0;left:-9999px;opacity:0'
    document.body.appendChild(ta)
    ta.select()
    ta.setSelectionRange(0, ta.value.length)
    return document.execCommand('copy')
  } catch (e) {
    console.warn('execCommand 复制失败', e)
    return false
  } finally {
    if (ta && ta.parentNode) ta.parentNode.removeChild(ta)
  }
}
