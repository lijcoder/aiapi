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

    <!-- 两步验证（2FA） -->
    <n-card title="两步验证（2FA）" size="small">
      <div style="display:flex;align-items:center;gap:12px;margin-top:4px">
        <n-tag :type="user?.totp_enabled ? 'success' : 'default'" size="small">
          {{ user?.totp_enabled ? '已开启' : '未开启' }}
        </n-tag>
        <span style="color:#666;font-size:13px">
          {{ user?.totp_enabled ? '登录时除密码外还需输入 Authenticator 验证码' : '开启后登录需额外输入 Authenticator 验证码，提升账号安全性' }}
        </span>
        <n-button
          :type="user?.totp_enabled ? 'error' : 'primary'"
          size="small"
          style="margin-left:auto"
          @click="user?.totp_enabled ? (showDisable = true) : openSetup()"
        >
          {{ user?.totp_enabled ? '关闭' : '开启' }}
        </n-button>
      </div>
    </n-card>

    <!-- 开启 2FA：扫码 + 首个验证码确认 -->
    <n-modal v-model:show="showSetup" preset="card" title="开启两步验证" style="width:420px" :bordered="false">
      <n-spin :show="setupLoading">
        <div v-if="setup" style="display:flex;flex-direction:column;align-items:center;gap:12px">
          <p style="color:#666;font-size:13px;margin:0;text-align:center">
            使用 Google Authenticator / 1Password / 微软 Authenticator 扫描二维码
          </p>
          <img :src="setup.qr_code" alt="TOTP 二维码" style="width:200px;height:200px" />
          <div style="font-size:12px;color:#999;text-align:center">
            无法扫码？手动输入密钥：<code style="color:#666;user-select:all">{{ setup.secret }}</code>
          </div>
          <n-input v-model:value="setupCode" placeholder="输入 App 中显示的 6 位验证码" maxlength="6" :allow-input="onlyDigits" style="max-width:240px" />
          <p v-if="setupMsg" style="color:#d03050;font-size:13px;margin:0">{{ setupMsg }}</p>
          <n-button type="primary" :loading="confirming" block @click="confirmSetup">确认绑定</n-button>
        </div>
      </n-spin>
    </n-modal>

    <!-- 关闭 2FA：需校验密码 -->
    <n-modal v-model:show="showDisable" preset="card" title="关闭两步验证" style="width:400px" :bordered="false">
      <div style="display:flex;flex-direction:column;gap:12px">
        <p style="color:#666;font-size:13px;margin:0">关闭后登录只需账号密码，安全性降低。请输入登录密码确认。</p>
        <n-input v-model:value="disablePassword" type="password" placeholder="登录密码" show-password-on="click" />
        <p v-if="disableMsg" style="color:#d03050;font-size:13px;margin:0">{{ disableMsg }}</p>
        <n-button type="error" :loading="disabling" block @click="confirmDisable">确认关闭</n-button>
      </div>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NInput, NButton, NForm, NFormItem, NTag, NModal, NSpin, useMessage } from 'naive-ui'
import { updateProfileSelf, setup2faSelf, confirm2faSelf, disable2faSelf } from '../api'
import { useUser } from '../stores/user'

const message = useMessage()
const router = useRouter()
const { user, fetchUser } = useUser()

const onlyDigits = v => /^\d*$/.test(v)

const profileForm = ref({ name: '', email: '' })
const profileMsg = ref('')
const savingProfile = ref(false)

// ===== 2FA 绑定 =====
const showSetup = ref(false)
const setupLoading = ref(false)
const setup = ref(null)
const setupCode = ref('')
const setupMsg = ref('')
const confirming = ref(false)

// ===== 2FA 关闭 =====
const showDisable = ref(false)
const disablePassword = ref('')
const disableMsg = ref('')
const disabling = ref(false)

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

async function openSetup() {
  showSetup.value = true
  setupLoading.value = true
  setup.value = null
  setupCode.value = ''
  setupMsg.value = ''
  try {
    setup.value = await setup2faSelf()
  } catch (e) {
    showSetup.value = false
    message.error(e.msg || '生成二维码失败')
  } finally {
    setupLoading.value = false
  }
}

async function confirmSetup() {
  setupMsg.value = ''
  if (!/^\d{6}$/.test(setupCode.value)) {
    setupMsg.value = '请输入 6 位数字验证码'
    return
  }
  confirming.value = true
  try {
    await confirm2faSelf(setup.value.setup_ticket, setupCode.value)
    await fetchUser()
    showSetup.value = false
    message.success('两步验证已开启')
  } catch (e) {
    setupMsg.value = e.msg || '绑定失败'
  } finally {
    confirming.value = false
  }
}

async function confirmDisable() {
  disableMsg.value = ''
  if (!disablePassword.value) {
    disableMsg.value = '请输入登录密码'
    return
  }
  disabling.value = true
  try {
    await disable2faSelf(disablePassword.value)
    await fetchUser()
    showDisable.value = false
    disablePassword.value = ''
    message.success('两步验证已关闭')
  } catch (e) {
    disableMsg.value = e.msg || '关闭失败'
  } finally {
    disabling.value = false
  }
}
</script>
