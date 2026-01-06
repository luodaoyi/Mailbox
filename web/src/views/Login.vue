<template>
  <div class="flex items-center justify-center min-h-screen">
    <div class="w-full max-w-md p-8 bg-white rounded-lg shadow">
      <h1 class="text-2xl font-bold mb-6 text-center">临时邮箱</h1>
      <form @submit.prevent="handleLogin">
        <input v-model="email" type="email" placeholder="输入邮箱地址 (如 test@cc.com)"
               class="w-full px-4 py-3 mb-4 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" required>
        <button type="submit" :disabled="loading"
                class="w-full px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50">
          {{ loading ? '登录中...' : '进入邮箱' }}
        </button>
      </form>
      <p v-if="error" class="mt-4 text-red-500 text-center">{{ error }}</p>
      <p class="mt-6 text-center text-gray-500 text-sm">
        <router-link to="/admin" class="text-blue-500 hover:underline">管理员入口</router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const email = ref('')
const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await api.login(email.value)
    localStorage.setItem('token', res.token)
    localStorage.setItem('email', email.value)
    router.push('/inbox')
  } catch (e) {
    error.value = e.response?.data?.error || '域名未启用或登录失败'
  } finally {
    loading.value = false
  }
}
</script>
