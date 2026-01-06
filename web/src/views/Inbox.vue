<template>
  <div class="max-w-4xl mx-auto p-4">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold">收件箱 - {{ email }}</h1>
      <button @click="logout" class="px-4 py-2 text-gray-600 hover:text-gray-800">退出</button>
    </div>
    <div class="bg-white rounded-lg shadow">
      <div v-if="loading" class="p-8 text-center text-gray-500">加载中...</div>
      <div v-else-if="emails.length === 0" class="p-8 text-center text-gray-500">暂无邮件</div>
      <div v-else>
        <div v-for="e in emails" :key="e.ID" @click="router.push(`/email/${e.ID}`)"
             class="p-4 border-b cursor-pointer hover:bg-gray-50">
          <div class="flex justify-between">
            <span class="font-medium truncate">{{ e.Subject || '(无主题)' }}</span>
            <span class="text-sm text-gray-500">{{ formatTime(e.ReceivedAt) }}</span>
          </div>
          <div class="text-sm text-gray-600 truncate">{{ e.FromAddr }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const email = ref(localStorage.getItem('email') || '')
const emails = ref([])
const loading = ref(true)
let eventSource = null

const formatTime = t => new Date(t).toLocaleString('zh-CN')

const logout = () => {
  eventSource?.close()
  localStorage.removeItem('token')
  localStorage.removeItem('email')
  router.push('/')
}

const connectSSE = () => {
  const token = localStorage.getItem('token')
  if (!token) return
  eventSource = new EventSource(`/api/emails/stream?token=${token}`)
  eventSource.onmessage = e => {
    const newEmail = JSON.parse(e.data)
    emails.value.unshift(newEmail)
  }
  eventSource.onerror = () => {
    eventSource?.close()
    setTimeout(connectSSE, 3000)
  }
}

onMounted(async () => {
  try {
    emails.value = await api.getEmails()
    connectSSE()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})

onUnmounted(() => eventSource?.close())
</script>
