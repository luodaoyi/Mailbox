<template>
  <div class="h-screen flex flex-col">
    <div class="bg-white border-b px-6 py-4 flex justify-between items-center">
      <h1 class="text-xl font-bold">收件箱 - {{ email }}</h1>
      <div class="flex gap-4">
        <router-link to="/admin" class="px-4 py-2 text-blue-600 hover:text-blue-800">管理后台</router-link>
        <button @click="logout" class="px-4 py-2 text-gray-600 hover:text-gray-800">退出</button>
      </div>
    </div>
    <div class="flex-1 flex overflow-hidden">
      <!-- 左侧邮件列表 -->
      <div class="w-96 border-r bg-white overflow-y-auto">
        <div v-if="loading" class="p-8 text-center text-gray-500">加载中...</div>
        <div v-else-if="emails.length === 0" class="p-8 text-center text-gray-500">暂无邮件</div>
        <div v-else>
          <div v-for="e in emails" :key="e.ID"
               :class="['p-4 border-b hover:bg-gray-50 flex justify-between items-start', selectedEmail?.ID === e.ID ? 'bg-blue-50' : '']">
            <div class="flex-1 cursor-pointer" @click="selectEmail(e)">
              <div class="flex justify-between mb-1">
                <span class="font-medium truncate">{{ e.Subject || '(无主题)' }}</span>
                <span class="text-xs text-gray-500">{{ formatTime(e.ReceivedAt) }}</span>
              </div>
              <div class="text-sm text-gray-600 truncate">{{ e.FromAddr }}</div>
            </div>
            <button @click.stop="deleteEmail(e.ID)" class="ml-2 text-red-500 hover:text-red-700" title="删除">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
      <!-- 右侧邮件详情 -->
      <div class="flex-1 bg-gray-50 overflow-y-auto">
        <div v-if="!selectedEmail" class="flex items-center justify-center h-full text-gray-500">
          选择一封邮件查看详情
        </div>
        <div v-else class="bg-white m-4 rounded-lg shadow p-6">
          <h2 class="text-2xl font-bold mb-4">{{ selectedEmail.subject || '(无主题)' }}</h2>
          <div class="text-sm text-gray-600 mb-4 space-y-1">
            <div><span class="font-medium">发件人:</span> {{ selectedEmail.from_addr }}</div>
            <div><span class="font-medium">收件人:</span> {{ selectedEmail.to_addr }}</div>
            <div><span class="font-medium">时间:</span> {{ formatTime(selectedEmail.received_at) }}</div>
          </div>
          <div v-if="selectedEmail.attachments?.length" class="mb-4 p-3 bg-gray-50 rounded">
            <div class="font-medium mb-2">附件 ({{ selectedEmail.attachments.length }}):</div>
            <div v-for="a in selectedEmail.attachments" :key="a.id" class="text-sm">
              <a :href="api.getAttachment(a.id)" target="_blank" class="text-blue-500 hover:underline">
                {{ a.filename }} ({{ formatSize(a.size) }})
              </a>
            </div>
          </div>
          <div class="border-t pt-4">
            <div v-if="selectedEmail.html_body" v-html="selectedEmail.html_body" class="prose max-w-none"></div>
            <pre v-else class="whitespace-pre-wrap">{{ selectedEmail.body }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import api from '../api'

const router = useRouter()
const route = useRoute()
const email = ref('')
const emails = ref([])
const selectedEmail = ref(null)
const loading = ref(true)
let eventSource = null
let pollTimer = null

const formatTime = t => new Date(t).toLocaleString('zh-CN')
const formatSize = s => s < 1024 ? s + 'B' : s < 1048576 ? (s / 1024).toFixed(1) + 'KB' : (s / 1048576).toFixed(1) + 'MB'

const logout = () => {
  eventSource?.close()
  if (pollTimer) clearInterval(pollTimer)
  localStorage.removeItem('token')
  router.push('/')
}

const selectEmail = async (e) => {
  try {
    selectedEmail.value = await api.getEmail(e.ID)
  } catch (err) {
    console.error(err)
  }
}

const deleteEmail = async (id) => {
  if (!confirm('确定删除这封邮件吗？')) return
  try {
    await api.deleteEmail(id)
    emails.value = emails.value.filter(e => e.ID !== id)
    if (selectedEmail.value?.id === id) {
      selectedEmail.value = null
    }
  } catch (err) {
    console.error('删除邮件失败:', err)
    console.error('错误详情:', err.response?.data)
    alert('删除失败: ' + (err.response?.data?.error || err.message))
  }
}

const loadEmails = async () => {
  try {
    const data = await api.getEmails()
    emails.value = data
  } catch (e) {
    console.error(e)
  }
}

const connectSSE = () => {
  const token = localStorage.getItem('token')
  if (!token) return
  eventSource = new EventSource(`/api/emails/stream?token=${token}`)
  eventSource.onmessage = e => {
    const newEmail = JSON.parse(e.data)
    if (!emails.value.find(em => em.ID === newEmail.ID)) {
      emails.value.unshift(newEmail)
    }
  }
  eventSource.onerror = () => {
    eventSource?.close()
    setTimeout(connectSSE, 3000)
  }
}

const startPolling = () => {
  pollTimer = setInterval(async () => {
    try {
      const data = await api.getEmails()
      data.forEach(newEmail => {
        if (!emails.value.find(e => e.ID === newEmail.ID)) {
          emails.value.unshift(newEmail)
        }
      })
    } catch (e) {
      console.error('Polling error:', e)
    }
  }, 30000)
}

onMounted(async () => {
  email.value = route.query.email
  if (!email.value) {
    router.push('/')
    return
  }

  let token = localStorage.getItem('token')
  if (!token) {
    try {
      const res = await api.login(email.value)
      token = res.token
      localStorage.setItem('token', token)
    } catch (err) {
      console.error('自动登录失败:', err)
      router.push('/')
      return
    }
  }

  await loadEmails()
  loading.value = false
  connectSSE()
  startPolling()
})

onUnmounted(() => {
  eventSource?.close()
  if (pollTimer) clearInterval(pollTimer)
})
</script>
