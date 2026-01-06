import axios from 'axios'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      location.href = '/'
    }
    return Promise.reject(err)
  }
)

export default {
  login: email => api.post('/login', { email }),
  getEmails: () => api.get('/emails'),
  getEmail: id => api.get(`/emails/${id}`),
  getAttachment: id => `/api/attachments/${id}`,

  adminLogin: (username, password) => api.post('/admin/login', { username, password }),
  getDomains: () => api.get('/admin/domains'),
  addDomain: domain => api.post('/admin/domains', { domain, enabled: true }),
  updateDomain: (id, enabled) => api.put(`/admin/domains/${id}`, { enabled }),
  deleteDomain: id => api.delete(`/admin/domains/${id}`)
}
