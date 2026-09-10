import { sitePath } from './utils/paths'

export async function api(path, options = {}) {
  const headers = { 'X-Requested-With': 'ADN', ...options.headers }
  let body = options.body
  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(body)
  }
  const response = await fetch(sitePath(path), { ...options, body, headers })
  let result
  try {
    result = await response.json()
  } catch {
    throw new Error('服务返回异常，请稍后重试')
  }
  if (!response.ok) {
    const error = new Error(result.errors?.join('\n') || result.error || '请求失败')
    error.status = response.status
    if (response.status === 401 && path !== '/api/me' && path !== '/api/auth/local')
      location.reload()
    throw error
  }
  return result
}
