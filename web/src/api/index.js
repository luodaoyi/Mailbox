import axios from 'axios'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use(config => {
  // 只为管理后台 API 添加 token
  const isAdmin = config.url.startsWith('/admin')
  if (isAdmin) {
    const token = localStorage.getItem('adminToken')
    if (token) config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      const isAdmin = err.config.url.startsWith('/admin')
      const tokenKey = isAdmin ? 'adminToken' : 'userToken'
      const token = localStorage.getItem(tokenKey)
      if (token) {
        localStorage.removeItem(tokenKey)
        if (isAdmin) {
          location.href = '/admin'
        } else {
          location.href = '/'
        }
      }
    }
    return Promise.reject(err)
  }
)

export default {
  login: email => api.post('/login', { email }),
  getEmails: (email, page = 1, limit = 50) => api.get('/emails', { params: { email, page, limit } }),
  getEmail: (email, id) => api.get(`/emails/${id}`, { params: { email } }),
  getAttachment: (email, id) => `/api/attachments/${id}?email=${encodeURIComponent(email)}`,
  deleteEmail: (email, id) => api.delete(`/emails/${id}`, { params: { email } }),
  deletePageEmails: (email, page, limit) => api.delete('/emails/page', { params: { email, page, limit } }),
  deleteMonthEmails: (email) => api.delete('/emails/month/all', { params: { email } }),
  deleteAllEmails: (email) => api.delete('/emails/all', { params: { email } }),

  adminLogin: (username, password) => api.post('/admin/login', { username, password }),
  getDomains: () => api.get('/admin/domains'),
  addDomain: domain => api.post('/admin/domains', { domain, enabled: true }),
  updateDomain: (id, enabled) => api.put(`/admin/domains/${id}`, { enabled }),
  deleteDomain: id => api.delete(`/admin/domains/${id}`),
  getMailboxes: (domain = '', page = 1, limit = 50) => api.get('/admin/mailboxes', { params: { domain, page, limit } }),
  deleteMailbox: id => api.delete(`/admin/mailboxes/${id}`),
  deleteMailboxesByDomain: domain => api.delete(`/admin/mailboxes/domain/${domain}`)
}
