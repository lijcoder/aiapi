<template>
  <div style="display:flex;flex-direction:column;gap:16px;max-width:560px">
    <!-- 基本资料 -->
    <n-card title="基本资料" size="small">
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <div style="font-size:13px;margin-bottom:6px">账号</div>
          <n-input :value="user?.account" disabled />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">姓名</div>
          <n-input v-model:value="profileForm.name" placeholder="姓名" maxlength="64" />
        </div>
        <div>
          <div style="font-size:13px;margin-bottom:6px">邮箱（选填）</div>
          <n-input v-model:value="profileForm.email" placeholder="name@example.com" maxlength="128" />
        </div>
        <p v-if="profileMsg" style="color:#d03050;font-size:13px;margin:0">{{ profileMsg }}</p>
        <div style="display:flex;gap:10px">
          <n-button type="primary" :loading="savingProfile" @click="saveProfile">保存资料</n-button>
          <n-button type="error" @click="router.push('/profile/password')">修改密码</n-button>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NInput, NButton, useMessage } from 'naive-ui'
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
