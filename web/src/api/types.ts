/** Shared API list wrappers */
export interface Paginated<T> {
  list: T[]
  total: number
  page: number
  size: number
}

/** Backend Success() uses HTTP 200 body with code: 200 (not 0). */
export function isApiSuccess(code: number): boolean {
  return code === 200
}
