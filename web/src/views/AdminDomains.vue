<template>
  <div class="h-screen flex flex-col">
    <div class="bg-white border-b px-6 py-4 flex justify-between items-center">
      <h1 class="text-2xl font-bold">管理后台</h1>
      <button @click="logout" class="px-4 py-2 text-gray-600 hover:text-gray-800">退出</button>
    </div>

    <div class="flex-1 flex overflow-hidden">
      <!-- 左侧域名管理 -->
      <div class="w-96 border-r bg-white flex flex-col">
        <div class="p-4 border-b">
          <h2 class="text-xl font-bold mb-4">域名管理</h2>
          <div class="flex gap-2">
            <input v-model="newDomain" type="text" placeholder="输入域名 (如 cc.com)"
                   class="flex-1 px-4 py-2 border rounded-lg" @keyup.enter="handleAdd">
            <button @click="handleAdd" class="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">添加</button>
          </div>
        </div>
        <div class="flex-1 overflow-y-auto">
          <div v-if="loading" class="p-8 text-center text-gray-500">加载中...</div>
          <div v-else-if="domains.length === 0" class="p-8 text-center text-gray-500">暂无域名</div>
          <div v-else>
            <div v-for="d in domains" :key="d.ID"
                 :class="['p-4 border-b hover:bg-gray-50 cursor-pointer', selectedDomain?.ID === d.ID ? 'bg-blue-50' : '']"
                 @click="selectDomain(d)">
              <div class="flex justify-between items-center">
                <span class="font-medium">{{ d.Domain }}</span>
                <div class="flex gap-2">
                  <button @click.stop="toggleDomain(d)" :class="d.Enabled ? 'bg-green-500' : 'bg-gray-400'"
                          class="px-3 py-1 text-white rounded text-sm">
                    {{ d.Enabled ? '已启用' : '已禁用' }}
                  </button>
                  <button @click.stop="handleDelete(d.ID)" class="px-3 py-1 bg-red-500 text-white rounded text-sm hover:bg-red-600">
                    删除
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧邮箱列表 -->
      <div class="flex-1 bg-gray-50 flex flex-col">
        <div class="bg-white border-b p-4 flex justify-between items-center">
          <h2 class="text-xl font-bold">
            邮箱列表
            <span v-if="selectedDomain" class="text-blue-600"> - {{ selectedDomain.Domain }}</span>
            <span v-else class="text-gray-500"> - 全部域名</span>
          </h2>
          <button v-if="selectedDomain" @click="deleteAllMailboxes"
                  class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600">
            删除所有
          </button>
        </div>
        <div class="flex-1 overflow-y-auto p-4">
          <div v-if="mailboxLoading" class="flex items-center justify-center h-full text-gray-500">加载中...</div>
          <div v-else-if="mailboxes.length === 0" class="flex items-center justify-center h-full text-gray-500">暂无邮箱</div>
          <div v-else class="grid grid-cols-1 gap-4">
            <div v-for="m in mailboxes" :key="m.ID" class="bg-white rounded-lg shadow p-4">
              <div class="flex justify-between items-start">
                <div class="flex-1">
                  <div class="font-medium text-lg">{{ m.Address }}</div>
                  <div class="text-sm text-gray-500 mt-1">域名: {{ m.Domain }}</div>
                  <div class="text-sm text-gray-500">邮件数: {{ m.EmailCount }}</div>
                  <div class="text-sm text-gray-500">最后邮件: {{ formatTime(m.LastEmail) }}</div>
                </div>
                <div class="flex gap-2">
                  <a :href="`/inbox?email=${encodeURIComponent(m.Address)}`" target="_blank"
                     class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
                    查看
                  </a>
                  <button @click="deleteMailboxConfirm(m)"
                          class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600">
                    删除
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="mailboxTotal > mailboxLimit" class="bg-white border-t p-4 flex justify-between items-center">
          <button @click="prevMailboxPage" :disabled="mailboxPage === 1"
                  class="px-4 py-2 bg-gray-200 rounded disabled:opacity-50">上一页</button>
          <span class="text-sm text-gray-600">第 {{ mailboxPage }} 页 / 共 {{ totalMailboxPages }} 页 (共 {{ mailboxTotal }} 个)</span>
          <button @click="nextMailboxPage" :disabled="mailboxPage >= totalMailboxPages"
                  class="px-4 py-2 bg-gray-200 rounded disabled:opacity-50">下一页</button>
        </div>
      </div>
    </div>

    <!-- 删除进度模态框 -->
    <div v-if="deleting" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-xl font-bold mb-4">正在删除...</h3>
        <div class="flex items-center justify-center py-8">
          <svg class="animate-spin h-12 w-12 text-blue-500" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>
        <p class="text-center text-gray-600">请稍候，正在删除邮件和邮箱记录...</p>
      </div>
    </div>

    <!-- 确认对话框 -->
    <div v-if="confirmDialog.show" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-xl font-bold mb-4">{{ confirmDialog.title }}</h3>
        <p class="text-gray-700 whitespace-pre-line mb-6">{{ confirmDialog.message }}</p>
        <div class="flex gap-3 justify-end">
          <button @click="confirmDialog.show = false"
                  class="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300">
            取消
          </button>
          <button @click="confirmDialog.onConfirm"
                  class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600">
            确认
          </button>
        </div>
      </div>
    </div>

    <!-- 消息对话框 -->
    <div v-if="messageDialog.show" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 max-w-md w-full mx-4">
        <h3 class="text-xl font-bold mb-4">{{ messageDialog.title }}</h3>
        <p class="text-gray-700 whitespace-pre-line mb-6">{{ messageDialog.message }}</p>
        <div class="flex justify-end">
          <button @click="messageDialog.show = false"
                  class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
            确定
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const domains = ref([])
const selectedDomain = ref(null)
const newDomain = ref('')
const loading = ref(true)
const mailboxes = ref([])
const mailboxLoading = ref(true)
const mailboxPage = ref(1)
const mailboxLimit = ref(50)
const mailboxTotal = ref(0)
const deleting = ref(false)

const confirmDialog = ref({
  show: false,
  title: '',
  message: '',
  onConfirm: () => {}
})

const messageDialog = ref({
  show: false,
  title: '',
  message: ''
})

const showConfirm = (title, message, onConfirm) => {
  confirmDialog.value = {
    show: true,
    title,
    message,
    onConfirm: () => {
      confirmDialog.value.show = false
      onConfirm()
    }
  }
}

const showMessage = (title, message) => {
  messageDialog.value = {
    show: true,
    title,
    message
  }
}

const totalMailboxPages = computed(() => Math.ceil(mailboxTotal.value / mailboxLimit.value))

const formatTime = t => new Date(t).toLocaleString('zh-CN')

const selectDomain = (domain) => {
  selectedDomain.value = domain
  mailboxPage.value = 1
  loadMailboxes()
}

const deleteMailboxConfirm = async (mailbox) => {
  showConfirm(
    '确认删除',
    `确定删除邮箱 ${mailbox.Address} 及其所有邮件吗？\n\n该邮箱有 ${mailbox.EmailCount} 封邮件，此操作不可恢复！`,
    async () => {
      deleting.value = true
      try {
        const res = await api.deleteMailbox(mailbox.ID)
        showMessage('删除成功', `删除了 ${res.email_count} 封邮件`)
        loadMailboxes()
      } catch (e) {
        console.error(e)
        showMessage('删除失败', e.response?.data?.error || e.message)
      } finally {
        deleting.value = false
      }
    }
  )
}

const deleteAllMailboxes = async () => {
  if (!selectedDomain.value) return

  showConfirm(
    '确认删除所有',
    `确定删除域名 ${selectedDomain.value.Domain} 下的所有邮箱及邮件吗？\n\n当前有 ${mailboxTotal.value} 个邮箱，此操作不可恢复！\n\n请再次确认此操作。`,
    async () => {
      deleting.value = true
      try {
        const res = await api.deleteMailboxesByDomain(selectedDomain.value.Domain)
        showMessage('删除成功', `删除了 ${res.mailbox_count} 个邮箱\n删除了 ${res.email_count} 封邮件`)
        selectedDomain.value = null
        loadMailboxes()
      } catch (e) {
        console.error(e)
        showMessage('删除失败', e.response?.data?.error || e.message)
      } finally {
        deleting.value = false
      }
    }
  )
}

const loadDomains = async () => {
  try {
    domains.value = await api.getDomains()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const loadMailboxes = async () => {
  mailboxLoading.value = true
  try {
    const domain = selectedDomain.value?.Domain || ''
    const data = await api.getMailboxes(domain, mailboxPage.value, mailboxLimit.value)
    mailboxes.value = data.mailboxes || []
    mailboxTotal.value = data.total || 0
  } catch (e) {
    console.error(e)
  } finally {
    mailboxLoading.value = false
  }
}

const prevMailboxPage = () => {
  if (mailboxPage.value > 1) {
    mailboxPage.value--
    loadMailboxes()
  }
}

const nextMailboxPage = () => {
  if (mailboxPage.value < totalMailboxPages.value) {
    mailboxPage.value++
    loadMailboxes()
  }
}

const handleAdd = async () => {
  if (!newDomain.value.trim()) return
  try {
    await api.addDomain(newDomain.value.trim())
    newDomain.value = ''
    loadDomains()
  } catch (e) {
    showMessage('添加失败', e.response?.data?.error || e.message)
  }
}

const toggleDomain = async d => {
  try {
    await api.updateDomain(d.ID, !d.Enabled)
    loadDomains()
  } catch (e) {
    showMessage('更新失败', e.response?.data?.error || e.message)
  }
}

const handleDelete = async id => {
  showConfirm(
    '确认删除',
    '确定删除此域名吗？',
    async () => {
      try {
        await api.deleteDomain(id)
        loadDomains()
      } catch (e) {
        showMessage('删除失败', e.response?.data?.error || e.message)
      }
    }
  )
}

const logout = () => {
  localStorage.removeItem('adminToken')
  router.push('/admin')
}

onMounted(() => {
  const token = localStorage.getItem('adminToken')
  if (!token) {
    router.push('/admin')
    return
  }
  loadDomains()
  loadMailboxes()
})
</script>
