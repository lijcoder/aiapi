<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <div id="app">
          <router-view />
        </div>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { NMessageProvider, NConfigProvider, NDialogProvider, zhCN, dateZhCN } from 'naive-ui'
</script>
