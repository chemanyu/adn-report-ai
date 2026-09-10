// Relative production URLs work both at / and behind a reverse-proxy prefix.
export function sitePath(path = '/') {
  return import.meta.env.BASE_URL + path.replace(/^\//, '')
}
