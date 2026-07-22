<template>
  <div style="display:flex;flex-direction:column;gap:16px;max-width:600px">
    <!-- 基本资料 -->
    <n-card title="基本资料" size="small">
      <n-form label-placement="left" :label-width="80" size="medium" style="margin-top:8px">
        <n-form-item label="账号">
          <n-input :value="user?.account" disabled />
        </n-form-item>
        <n-form-item label="姓名">
          <n-input v-model:value="profileForm.name" placeholder="请输入姓名" maxlength="64" />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input v-model:value="profileForm.email" placeholder="name@example.com（选填）" maxlength="128" />
        </n-form-item>
      </n-form>

      <p v-if="profileMsg" style="color:#d03050;font-size:13px;margin:0 0 8px">{{ profileMsg }}</p>

      <div style="display:flex;padding-left:80px">
        <n-button type="primary" :loading="savingProfile" @click="saveProfile">保存资料</n-button>
        <n-button type="error" style="margin-left:auto" @click="router.push('/profile/password')">修改密码</n-button>
      </div>
    </n-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NInput, NButton, NForm, NFormItem, useMessage } from 'naive-ui'
import { updateProfileSelf } from '../api'
import { useUser } from '../stores/user'

const message = useMessage()
const router = useRouter()
const { user, fetchUser } = useUser()

const profileForm = ref({ name: '', email: '' })
const profileMsg = ref('')
const savingProfile = ref(false)

onMounted(() => {
  if (user.value) {
    profileForm.value.name = user.value.name || ''
    profileForm.value.email = user.value.email || ''
  }
})

async function saveProfile() {
  profileMsg.value = ''
  const name = profileForm.value.name.trim()
  if (!name) {
    profileMsg.value = '姓名不能为空'
    return
  }
  savingProfile.value = true
  try {
    await updateProfileSelf(name, profileForm.value.email.trim())
    await fetchUser()
    message.success('资料已保存')
  } catch (e) {
    profileMsg.value = e.msg || '保存失败'
  } finally {
    savingProfile.value = false
  }
}
</script>
