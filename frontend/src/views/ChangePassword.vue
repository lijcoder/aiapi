<template>
  <div style="max-width:480px">
    <n-card size="small">
      <template #header>
        <n-breadcrumb>
          <n-breadcrumb-item @click="router.push('/profile')">个人设置</n-breadcrumb-item>
          <n-breadcrumb-item>修改密码</n-breadcrumb-item>
        </n-breadcrumb>
      </template>
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">当前密码</div>
          <n-input v-model:value="pwdForm.old" type="password" show-password-on="click" placeholder="当前密码" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">新密码</div>
          <n-input v-model:value="pwdForm.new" type="password" show-password-on="click" placeholder="新密码" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">确认新密码</div>
          <n-input v-model:value="pwdForm.confirm" type="password" show-password-on="click" placeholder="再次输入新密码" maxlength="64" />
        </div>
        <p v-if="pwdMsg" style="color:#d03050;font-size:13px;margin:0">{{ pwdMsg }}</p>
        <n-space>
          <n-button @click="router.push('/profile')">返回</n-button>
          <n-button type="primary" :loading="savingPwd" @click="savePassword">修改密码</n-button>
        </n-space>
      </div>
    </n-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NBreadcrumb, NBreadcrumbItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { updatePasswordSelf } from '../api'

const message = useMessage()
const router = useRouter()

const pwdForm = ref({ old: '', new: '', confirm: '' })
const pwdMsg = ref('')
const savingPwd = ref(false)

async function savePassword() {
  pwdMsg.value = ''
  if (!pwdForm.value.old || !pwdForm.value.new) {
    pwdMsg.value = '请填写当前密码和新密码'
    return
  }
  if (pwdForm.value.new !== pwdForm.value.confirm) {
    pwdMsg.value = '两次输入的新密码不一致'
    return
  }
  savingPwd.value = true
  try {
    await updatePasswordSelf(pwdForm.value.old, pwdForm.value.new)
    message.success('密码已修改，请重新登录')
    setTimeout(() => {
      router.replace('/login')
    }, 1200)
  } catch (e) {
    pwdMsg.value = e.msg || '修改失败'
  } finally {
    savingPwd.value = false
  }
}
</script>
