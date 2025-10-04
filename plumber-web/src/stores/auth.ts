import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin, type LoginParams } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const username = ref<string>(localStorage.getItem('username') || '')
  const userId = ref<string>(localStorage.getItem('userId') || '')

  const isLoggedIn = computed(() => !!token.value)

  async function login(params: LoginParams) {
    const result = await apiLogin(params)

    token.value = result.token
    username.value = result.username
    userId.value = result.user_id

    localStorage.setItem('token', result.token)
    localStorage.setItem('username', result.username)
    localStorage.setItem('userId', result.user_id)

    return result
  }

  function logout() {
    token.value = ''
    username.value = ''
    userId.value = ''

    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('userId')
  }

  return {
    token,
    username,
    userId,
    isLoggedIn,
    login,
    logout,
  }
})
