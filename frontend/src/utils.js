// fix4 格式化金额：最多4位小数，去掉尾部无意义的 0
// 0.0000 → "0", 0.1000 → "0.1", 0.1234 → "0.1234"
export function fix4(n) {
  if (n == null) return '-'
  return Number(n).toFixed(4).replace(/\.?0+$/, '')
}
