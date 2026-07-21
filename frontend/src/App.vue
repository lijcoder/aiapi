<template>
  <n-config-provider>
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
import { NMessageProvider, NConfigProvider, NDialogProvider } from 'naive-ui'
</script>
