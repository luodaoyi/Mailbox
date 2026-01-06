<template>
  <div class="max-w-4xl mx-auto p-4">
    <button @click="router.push('/inbox')" class="mb-4 px-4 py-2 bg-gray-200 rounded hover:bg-gray-300">← 返回</button>
    <div v-if="loading" class="text-center py-8">加载中...</div>
    <div v-else-if="email" class="bg-white rounded-lg shadow p-6">
      <h1 class="text-xl font-bold mb-4">{{ email.Subject || '(无主题)' }}</h1>
      <div class="text-sm text-gray-600 mb-2">发件人: {{ email.FromAddr }}</div>
      <div class="text-sm text-gray-600 mb-4">时间: {{ formatTime(email.ReceivedAt) }}</div>
      <div v-if="email.Attachments?.length" class="mb-4 p-3 bg-gray-50 rounded">
        <div class="font-medium mb-2">附件:</div>
        <div v-for="a in email.Attachments" :key="a.ID" class="flex items-center gap-2">
          <a :href="api.getAttachment(a.ID)" target="_blank" class="text-blue-500 hover:underline">
            {{ a.Filename }} ({{ formatSize(a.Size) }})
          </a>
        </div>
      </div>
      <div class="border-t pt-4">
        <div v-if="email.HtmlBody" v-html="email.HtmlBody" class="prose max-w-none"></div>
        <pre v-else class="whitespace-pre-wrap">{{ email.Body }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import api from '../api'

const router = useRouter()
const route = useRoute()
const email = ref(null)
const loading = ref(true)

const formatTime = t => new Date(t).toLocaleString('zh-CN')
const formatSize = s => s < 1024 ? s + 'B' : (s / 1024).toFixed(1) + 'KB'

onMounted(async () => {
  try {
    email.value = await api.getEmail(route.params.id)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})
</script>
