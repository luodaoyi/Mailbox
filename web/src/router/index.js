import { createRouter, createWebHistory } from 'vue-router'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('../views/Login.vue') },
    { path: '/inbox', component: () => import('../views/Inbox.vue') },
    { path: '/email/:id', component: () => import('../views/EmailDetail.vue') },
    { path: '/admin', component: () => import('../views/AdminLogin.vue') },
    { path: '/admin/domains', component: () => import('../views/AdminDomains.vue') }
  ]
})
