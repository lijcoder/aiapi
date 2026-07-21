import { ref, computed } from 'vue'
import { self } from '../api'

const user = ref(null)
const permissions = ref([])
const menus = ref([])

export function useUser() {
  const isLoggedIn = computed(() => user.value !== null)

  function hasPath(path) {
    return permissions.value.some(p => p.entity === 'API' && p.value === path)
  }

  async function fetchUser() {
    const data = await self()
    user.value = data.user
    permissions.value = data.permissions
    menus.value = data.menus || []
  }

  function clearUser() {
    user.value = null
    permissions.value = []
    menus.value = []
  }

  return { user, permissions, menus, isLoggedIn, hasPath, fetchUser, clearUser }
}
