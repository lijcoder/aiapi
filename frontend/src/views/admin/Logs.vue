<template>
  <div style="display:flex;flex-direction:column;gap:16px">
    <!-- 操作栏 -->
    <n-card size="small">
      <div style="display:flex;align-items:center;gap:12px">
        <n-button type="primary" :loading="loading" @click="loadLogs" size="small">
          <template #icon>
            <svg viewBox="0 0 512 512" width="16" height="16" fill="currentColor">
              <path d="M320 146s24.36-12-64-12a160 160 0 10160 160" fill="none" stroke="currentColor" stroke-linecap="round" stroke-miterlimit="10" stroke-width="32"/>
              <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M256 58l80 80-80 80"/>
            </svg>
          </template>
          刷新
        </n-button>
        <span style="font-size:13px;color:#666">
          共 {{ logs.length }} 条日志（最近 {{ logs.length }} 条，倒序展示）
        </span>
        <div style="margin-left:auto">
          <n-button size="small" @click="scrollToTop" :disabled="logs.length === 0">回到顶部</n-button>
          <n-button size="small" style="margin-left:8px" @click="scrollToBottom" :disabled="logs.length === 0">回到底部</n-button>
        </div>
      </div>
    </n-card>

    <!-- 日志内容 -->
    <n-card size="small" :bordered="true">
      <div ref="logContainer" style="max-height:calc(100vh - 240px);overflow-y:auto;background:#fafafa;border:1px solid #e8e8e8;border-radius:4px;font-family:'SF Mono','Fira Code','Menlo','Consolas',monospace;font-size:12px;line-height:1.6">
        <div v-if="loading" style="padding:40px;text-align:center;color:#999;font-size:14px">加载中...</div>
        <div v-else-if="logs.length === 0" style="padding:40px;text-align:center;color:#999;font-size:14px">暂无日志</div>
        <div v-else style="padding:8px 0">
          <div v-for="(line, idx) in logs" :key="idx" style="display:flex;align-items:flex-start;padding:1px 12px;transition:background 0.1s" :style="idx % 2 === 0 ? { background: '#fafafa' } : { background: '#f5f5f5' }">
            <span style="min-width:36px;color:#b0b0b0;text-align:right;margin-right:12px;user-select:none;flex-shrink:0">{{ idx + 1 }}</span>
            <span style="color:#333;word-break:break-all;white-space:pre-wrap">{{ line }}</span>
          </div>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { NCard, NButton, useMessage } from 'naive-ui'
import { fetchLogs } from '../../api'

const message = useMessage()
const loading = ref(false)
const logs = ref([])
const logContainer = ref(null)

async function loadLogs() {
  loading.value = true
  try {
    const data = await fetchLogs()
    logs.value = data.lines || []
  } catch (e) {
    message.error(e.msg || '加载日志失败')
  } finally {
    loading.value = false
  }
}

function scrollToTop() {
  if (logContainer.value) {
    logContainer.value.scrollTop = 0
  }
}

function scrollToBottom() {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

onMounted(() => {
  loadLogs()
})
</script>
