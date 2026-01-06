<template>
  <div class="max-w-4xl mx-auto p-4">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold">域名管理</h1>
      <div class="flex gap-4">
        <router-link to="/inbox" class="px-4 py-2 text-blue-600 hover:text-blue-800">邮件列表</router-link>
        <button @click="logout" class="px-4 py-2 text-gray-600 hover:text-gray-800">退出</button>
      </div>
    </div>
    <div class="mb-4 flex gap-2">
      <input v-model="newDomain" type="text" placeholder="输入域名 (如 cc.com)"
             class="flex-1 px-4 py-2 border rounded-lg" @keyup.enter="handleAdd">
      <button @click="handleAdd" class="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">添加</button>
    </div>
    <div class="bg-white rounded-lg shadow">
      <div v-if="loading" class="p-8 text-center text-gray-500">加载中...</div>
      <div v-else-if="domains.length === 0" class="p-8 text-center text-gray-500">暂无域名</div>
      <div v-else>
        <div v-for="d in domains" :key="d.ID" class="p-4 border-b flex justify-between items-center">
          <span>{{ d.Domain }}</span>
          <div class="flex gap-2">
            <button @click="toggleDomain(d)" :class="d.Enabled ? 'bg-green-500' : 'bg-gray-400'"
                    class="px-3 py-1 text-white rounded text-sm">
              {{ d.Enabled ? '已启用' : '已禁用' }}
            </button>
            <button @click="handleDelete(d.ID)" class="px-3 py-1 bg-red-500 text-white rounded text-sm hover:bg-red-600">
              删除
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const domains = ref([])
const newDomain = ref('')
const loading = ref(true)

const loadDomains = async () => {
  try {
    domains.value = await api.getDomains()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const handleAdd = async () => {
  if (!newDomain.value.trim()) return
  try {
    await api.addDomain(newDomain.value.trim())
    newDomain.value = ''
    loadDomains()
  } catch (e) {
    alert('添加失败')
  }
}

const toggleDomain = async d => {
  try {
    await api.updateDomain(d.ID, !d.Enabled)
    loadDomains()
  } catch (e) {
    alert('更新失败')
  }
}

const handleDelete = async id => {
  if (!confirm('确定删除?')) return
  try {
    await api.deleteDomain(id)
    loadDomains()
  } catch (e) {
    alert('删除失败')
  }
}

const logout = () => {
  localStorage.removeItem('token')
  router.push('/admin')
}

onMounted(() => {
  const token = localStorage.getItem('token')
  if (!token) {
    router.push('/admin')
    return
  }
  loadDomains()
})
</script>
