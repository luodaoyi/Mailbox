<template>
  <div class="flex items-center justify-center min-h-screen">
    <div class="w-full max-w-md p-8 bg-white rounded-lg shadow">
      <h1 class="text-2xl font-bold mb-6 text-center">管理员登录</h1>
      <form @submit.prevent="handleLogin">
        <input v-model="username" type="text" placeholder="用户名"
               class="w-full px-4 py-3 mb-4 border rounded-lg" required>
        <input v-model="password" type="password" placeholder="密码"
               class="w-full px-4 py-3 mb-4 border rounded-lg" required>
        <button type="submit" :disabled="loading"
                class="w-full px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
      <p v-if="error" class="mt-4 text-red-500 text-center">{{ error }}</p>
      <p class="mt-6 text-center text-gray-500 text-sm">
        <router-link to="/" class="text-blue-500 hover:underline">返回邮箱登录</router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await api.adminLogin(username.value, password.value)
    localStorage.setItem('adminToken', res.token)
    router.push('/admin/domains')
  } catch (e) {
    error.value = '用户名或密码错误'
  } finally {
    loading.value = false
  }
}
</script>
