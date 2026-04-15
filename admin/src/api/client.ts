import { get, post, put, del, patch } from '@/utils/request'

const DEFAULT_BASE = import.meta.env.VITE_API_BASE_URL || '/api'
const MAIN_API_BASE = import.meta.env.VITE_MAIN_API_BASE_URL || DEFAULT_BASE
const AUTH_API_BASE = import.meta.env.VITE_AUTH_API_BASE_URL || DEFAULT_BASE

const normalizeBase = (base: string): string => base.replace(/\/+$/, '')
const normalizePath = (path: string): string => (path.startsWith('/') ? path : `/${path}`)

export const buildMainApiUrl = (path: string): string => `${normalizeBase(MAIN_API_BASE)}${normalizePath(path)}`
export const buildAuthApiUrl = (path: string): string => `${normalizeBase(AUTH_API_BASE)}${normalizePath(path)}`

export const mainGet = <T = any>(path: string, config?: any) => get<T>(buildMainApiUrl(path), config)
export const mainPost = <T = any>(path: string, data?: any, config?: any) => post<T>(buildMainApiUrl(path), data, config)
export const mainPut = <T = any>(path: string, data?: any, config?: any) => put<T>(buildMainApiUrl(path), data, config)
export const mainDelete = <T = any>(path: string, config?: any) => del<T>(buildMainApiUrl(path), config)
export const mainPatch = <T = any>(path: string, data?: any, config?: any) => patch<T>(buildMainApiUrl(path), data, config)

export const authGet = <T = any>(path: string, config?: any) => get<T>(buildAuthApiUrl(path), config)
export const authPost = <T = any>(path: string, data?: any, config?: any) => post<T>(buildAuthApiUrl(path), data, config)
export const authPut = <T = any>(path: string, data?: any, config?: any) => put<T>(buildAuthApiUrl(path), data, config)
export const authDelete = <T = any>(path: string, config?: any) => del<T>(buildAuthApiUrl(path), config)
export const authPatch = <T = any>(path: string, data?: any, config?: any) => patch<T>(buildAuthApiUrl(path), data, config)

