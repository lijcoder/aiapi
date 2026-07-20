<template>
  <n-config-provider>
    <n-message-provider>
      <div id="app">
        <router-view />
      </div>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { NMessageProvider, NConfigProvider } from 'naive-ui'
</script>
