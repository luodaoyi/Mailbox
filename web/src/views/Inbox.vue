<template>
  <div class="h-screen flex flex-col">
    <div class="bg-white border-b px-6 py-4 flex justify-between items-center">
      <h1 class="text-xl font-bold">收件箱 - {{ email }}</h1>
      <div class="flex gap-2">
        <button @click="deletePageEmails" class="px-3 py-2 text-sm bg-yellow-500 text-white rounded hover:bg-yellow-600">删除本页</button>
        <button @click="deleteMonthEmails" class="px-3 py-2 text-sm bg-orange-500 text-white rounded hover:bg-orange-600">删除本月</button>
        <button @click="deleteAllEmails" class="px-3 py-2 text-sm bg-red-500 text-white rounded hover:bg-red-600">删除全部</button>
        <router-link to="/admin" class="px-4 py-2 text-blue-600 hover:text-blue-800">管理后台</router-link>
        <button @click="logout" class="px-4 py-2 text-gray-600 hover:text-gray-800">退出</button>
      </div>
    </div>
    <div class="flex-1 flex overflow-hidden">
      <!-- 左侧邮件列表 -->
      <div class="w-96 border-r bg-white flex flex-col">
        <div class="flex-1 overflow-y-auto">
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
        <div v-if="total > limit" class="border-t p-4 flex justify-between items-center">
          <button @click="prevPage" :disabled="page === 1" class="px-3 py-1 bg-gray-200 rounded disabled:opacity-50">上一页</button>
          <span class="text-sm text-gray-600">第 {{ page }} 页 / 共 {{ totalPages }} 页 (共 {{ total }} 封)</span>
          <button @click="nextPage" :disabled="page >= totalPages" class="px-3 py-1 bg-gray-200 rounded disabled:opacity-50">下一页</button>
        </div>
      </div>
      <!-- 右侧邮件详情 -->
      <div class="flex-1 bg-gray-50 overflow-y-auto">
        <div v-if="detailLoading" class="flex items-center justify-center h-full">
          <div class="text-gray-500">
            <svg class="animate-spin h-8 w-8 mx-auto mb-2" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p>加载中...</p>
          </div>
        </div>
        <div v-else-if="!selectedEmail" class="flex items-center justify-center h-full text-gray-500">
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
              <a :href="api.getAttachment(email, a.id)" target="_blank" class="text-blue-500 hover:underline">
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
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import api from '../api'

const router = useRouter()
const route = useRoute()
const email = ref('')
const emails = ref([])
const selectedEmail = ref(null)
const loading = ref(true)
const detailLoading = ref(false)
const page = ref(1)
const limit = ref(50)
const total = ref(0)
let eventSource = null
let pollTimer = null

const totalPages = computed(() => Math.ceil(total.value / limit.value))

const formatTime = t => new Date(t).toLocaleString('zh-CN')
const formatSize = s => s < 1024 ? s + 'B' : s < 1048576 ? (s / 1024).toFixed(1) + 'KB' : (s / 1048576).toFixed(1) + 'MB'

const prevPage = () => {
  if (page.value > 1) {
    page.value--
    loading.value = true
    loadEmails()
  }
}

const nextPage = () => {
  if (page.value < totalPages.value) {
    page.value++
    loading.value = true
    loadEmails()
  }
}

const logout = () => {
  eventSource?.close()
  if (pollTimer) clearInterval(pollTimer)
  router.push('/')
}

const selectEmail = async (e) => {
  detailLoading.value = true
  try {
    selectedEmail.value = await api.getEmail(email.value, e.ID)
  } catch (err) {
    console.error(err)
  } finally {
    detailLoading.value = false
  }
}

const deleteEmail = async (id) => {
  if (!confirm('确定删除这封邮件吗？')) return
  try {
    await api.deleteEmail(email.value, id)
    loadEmails()
    if (selectedEmail.value?.id === id) {
      selectedEmail.value = null
    }
  } catch (err) {
    console.error('删除邮件失败:', err)
    console.error('错误详情:', err.response?.data)
    alert('删除失败: ' + (err.response?.data?.error || err.message))
  }
}

const deletePageEmails = async () => {
  if (!confirm(`确定删除本页所有邮件吗？(共 ${emails.value.length} 封)`)) return
  try {
    const res = await api.deletePageEmails(email.value, page.value, limit.value)
    alert(`成功删除 ${res.count} 封邮件`)
    loadEmails()
    selectedEmail.value = null
  } catch (err) {
    console.error('删除失败:', err)
    alert('删除失败: ' + (err.response?.data?.error || err.message))
  }
}

const deleteMonthEmails = async () => {
  if (!confirm('确定删除本月所有邮件吗？此操作不可恢复！')) return
  try {
    const res = await api.deleteMonthEmails(email.value)
    alert(`成功删除 ${res.count} 封邮件`)
    page.value = 1
    loadEmails()
    selectedEmail.value = null
  } catch (err) {
    console.error('删除失败:', err)
    alert('删除失败: ' + (err.response?.data?.error || err.message))
  }
}

const deleteAllEmails = async () => {
  if (!confirm('确定删除所有邮件吗？此操作不可恢复！')) return
  if (!confirm('再次确认：真的要删除所有邮件吗？')) return
  try {
    const res = await api.deleteAllEmails(email.value)
    alert(`成功删除 ${res.count} 封邮件`)
    page.value = 1
    loadEmails()
    selectedEmail.value = null
  } catch (err) {
    console.error('删除失败:', err)
    alert('删除失败: ' + (err.response?.data?.error || err.message))
  }
}

const loadEmails = async () => {
  try {
    const data = await api.getEmails(email.value, page.value, limit.value)
    emails.value = data.emails || []
    total.value = data.total || 0
  } catch (e) {
    console.error(e)
    emails.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const connectSSE = () => {
  eventSource = new EventSource(`/api/emails/stream?email=${encodeURIComponent(email.value)}`)
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
    if (page.value === 1) {
      loadEmails()
    }
  }, 60000)
}

onMounted(async () => {
  email.value = route.query.email
  if (!email.value) {
    router.push('/')
    return
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
