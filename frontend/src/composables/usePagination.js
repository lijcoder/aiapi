import { ref, h } from 'vue'

/**
 * usePagination — n-data-table 远程分页通用逻辑
 * @param {(page:number, pageSize:number) => void} loadFn — 加载数据的函数
 * @param {number} [defaultPageSize=20] — 默认每页条数
 * @returns {{ pagination, onPage, onPageSize, resetAndLoad }}
 */
export function usePagination(loadFn, defaultPageSize = 20) {
  const pagination = ref({
    page: 1,
    pageSize: defaultPageSize,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100],
    prefix: () => h('span', { style: 'color:#909399;font-size:13px' }, `共 ${pagination.value.itemCount} 条`),
  })

  function onPage(p) {
    pagination.value.page = p
    loadFn()
  }

  function onPageSize(ps) {
    pagination.value.pageSize = ps
    pagination.value.page = 1
    loadFn()
  }

  function resetAndLoad() {
    pagination.value.page = 1
    loadFn()
  }

  return { pagination, onPage, onPageSize, resetAndLoad }
}
