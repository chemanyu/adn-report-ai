import { sitePath } from '../utils/paths'
import { reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { api } from '../api'
import { formatDecimal } from '../utils/format'

export function useWorkspace() {
  const state = reactive({
    ready: false,
    user: null,
    auth: {},
    login: { username: 'admin', password: '' },
    loginError: '',
    loggingIn: false,
    view: 'reports',
    accounts: [],
    filters: { q: '', account_id: '' },
    page: 1,
    records: { items: [], total: 0, row_count: 0, total_amount: '0', account_count: 0 },
    listError: '',
    listLoading: false,
    account: { name: '', code: '' },
    accountError: '',
    savingAccount: false,
    file: null,
    sheet: '',
    sheets: [],
    operator: '',
    accountID: '',
    preview: null,
    valid: false,
    previewLoading: false,
    uploadError: '',
    uploading: false,
    dragging: false,
    detailID: null,
    detailPage: 1,
    detail: null,
    detailError: '',
    detailLoading: false,
    toast: '',
  })
  let previewVersion = 0,
    listVersion = 0,
    detailVersion = 0,
    toastTimer
  // Native dialog focus/keyboard behavior is retained; data and rendering belong to Vue.
  const dialogs = {}
  function dialogRef(name, element) {
    dialogs[name] = element
  }
  function closeDialog(name) {
    if (name === 'upload' && state.uploading) return
    dialogs[name]?.close()
  }
  function toast(message) {
    state.toast = message
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => {
      state.toast = ''
    }, 4200)
  }
  const money = (value) => formatDecimal(value ?? '0', { grouping: true })
  const number = (value) => Number(value).toLocaleString('zh-CN')
  const datetime = (value) =>
    new Date(value).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  async function loadAccounts() {
    state.accounts = await api('/api/accounts')
  }
  async function loadRecords() {
    const version = ++listVersion
    state.listError = ''
    state.listLoading = true
    try {
      const data = await api(
        '/api/uploads?' + new URLSearchParams({ page: state.page, ...state.filters }),
      )
      if (version === listVersion) state.records = data
    } catch (e) {
      if (version === listVersion) state.listError = e.message
    } finally {
      if (version === listVersion) state.listLoading = false
    }
  }
  function filterRecords() {
    state.page = 1
    loadRecords()
  }
  function listPage(delta) {
    const page = state.page + delta
    if (page < 1 || page > Math.max(1, Math.ceil(state.records.total / 20))) return
    state.page = page
    loadRecords()
  }
  async function boot() {
    try {
      state.auth = await api('/api/auth/config')
      try {
        state.user = await api('/api/me')
      } catch (e) {
        if (e.status !== 401) throw e
      }
      if (state.user) {
        await loadAccounts()
        await loadRecords()
      }
    } catch (e) {
      if (state.user) state.listError = e.message
      else state.loginError = e.message
    } finally {
      state.ready = true
    }
    const error = new URLSearchParams(location.search).get('error')
    if (!state.user && error) {
      state.loginError =
        error === 'oauth_state'
          ? '授权已过期或不是当前浏览器发起，请重新登录。'
          : '钉钉授权失败，请重新登录并检查应用配置。'
      history.replaceState(null, '', sitePath('/'))
    }
  }
  async function login() {
    if (state.loggingIn) return
    state.loggingIn = true
    state.loginError = ''
    try {
      await api('/api/auth/local', { method: 'POST', body: state.login })
      location.href = sitePath('/')
    } catch (e) {
      state.loginError = e.message
    } finally {
      state.loggingIn = false
    }
  }
  async function logout() {
    try {
      await api('/api/auth/logout', { method: 'POST' })
      location.reload()
    } catch (e) {
      toast(e.message)
    }
  }
  function openAccount() {
    state.account = { name: '', code: '' }
    state.accountError = ''
    dialogs.account.showModal()
  }
  async function saveAccount() {
    if (state.savingAccount) return
    state.savingAccount = true
    state.accountError = ''
    try {
      const account = await api('/api/accounts', { method: 'POST', body: state.account })
      state.accounts = [account, ...state.accounts]
      state.accountID = String(account.id)
      closeDialog('account')
      toast('ADN 账户已保存')
    } catch (e) {
      state.accountError = e.message
    } finally {
      state.savingAccount = false
    }
  }
  async function openUpload() {
    previewVersion++
    Object.assign(state, {
      file: null,
      valid: false,
      preview: null,
      sheets: [],
      sheet: '',
      operator: state.user.name,
      accountID: '',
      uploadError: '',
      previewLoading: false,
      dragging: false,
    })
    dialogs.upload.querySelector('input[type=file]').value = ''
    await nextTick()
    dialogs.upload.showModal()
  }
  function formData() {
    const data = new FormData()
    data.append('file', state.file)
    data.append('sheet', state.sheet)
    data.append('operator', state.operator.trim())
    data.append('account_id', state.accountID)
    return data
  }
  async function preview() {
    const version = ++previewVersion
    state.valid = false
    state.preview = null
    state.uploadError = ''
    if (!state.file) {
      state.previewLoading = false
      return
    }
    state.previewLoading = true
    try {
      if (!/\.xlsx$/i.test(state.file.name) || state.file.size > 20 * 1024 * 1024)
        throw new Error('请选择 20 MB 以内的 .xlsx 文件。')
      const data = await api('/api/preview', { method: 'POST', body: formData() })
      if (version !== previewVersion) return
      state.preview = data
      state.valid = data.valid
      state.sheets = data.result.sheets
      state.sheet = data.result.sheet
    } catch (e) {
      if (version === previewVersion) state.uploadError = e.message
    } finally {
      if (version === previewVersion) state.previewLoading = false
    }
  }
  function chooseFile(file) {
    if (state.uploading) return
    state.file = file || null
    state.sheet = ''
    state.sheets = []
    preview()
  }
  function dropFile(event) {
    state.dragging = false
    if (state.uploading) return
    const file = event.dataTransfer.files[0]
    if (file) chooseFile(file)
  }
  async function upload() {
    if (!state.valid || !state.file || state.uploading) return
    state.uploading = true
    state.uploadError = ''
    try {
      const result = await api('/api/uploads', { method: 'POST', body: formData() })
      dialogs.upload.close()
      state.page = 1
      state.filters = { q: '', account_id: '' }
      state.view = 'reports'
      await loadRecords()
      toast(`上传成功，已保存 ${number(result.row_count)} 行结算数据`)
    } catch (e) {
      state.uploadError = e.message
    } finally {
      state.uploading = false
    }
  }
  async function loadDetail() {
    const version = ++detailVersion
    state.detailError = ''
    state.detailLoading = true
    state.detail = null
    try {
      const data = await api(`/api/uploads/${state.detailID}?page=${state.detailPage}`)
      if (version === detailVersion) state.detail = data
    } catch (e) {
      if (version === detailVersion) state.detailError = e.message
    } finally {
      if (version === detailVersion) state.detailLoading = false
    }
  }
  function openDetail(id) {
    state.detailID = id
    state.detailPage = 1
    dialogs.detail.showModal()
    loadDetail()
  }
  function detailPage(delta) {
    if (!state.detail || state.detailLoading) return
    const page = state.detailPage + delta
    if (page < 1 || page > Math.max(1, Math.ceil(state.detail.upload.row_count / 100))) return
    state.detailPage = page
    loadDetail()
  }
  onMounted(boot)
  onUnmounted(() => {
    clearTimeout(toastTimer)
    previewVersion++
    listVersion++
    detailVersion++
  })
  return {
    state,
    sitePath,
    dialogRef,
    closeDialog,
    money,
    number,
    datetime,
    login,
    logout,
    openAccount,
    saveAccount,
    openUpload,
    chooseFile,
    dropFile,
    preview,
    upload,
    filterRecords,
    listPage,
    openDetail,
    detailPage,
  }
}
