import { ref, computed } from 'vue'
import { self } from '../api'

const user = ref(null)
const permissions = ref([])

export function useUser() {
  const isLoggedIn = computed(() => user.value !== null)

  function hasPath(path) {
    return permissions.value.some(p => p.entity === 'API' && p.value === path)
  }

  async function fetchUser() {
    const data = await self()
    user.value = data.user
    permissions.value = data.permissions
  }

  function clearUser() {
    user.value = null
    permissions.value = []
  }

  return { user, permissions, isLoggedIn, hasPath, fetchUser, clearUser }
}
