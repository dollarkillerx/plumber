import axios, { type AxiosInstance } from 'axios'

// JSON-RPC 请求结构
export interface JSONRPCRequest {
  jsonrpc: '2.0'
  method: string
  params?: any
  id: string | number
}

// JSON-RPC 响应结构
export interface JSONRPCResponse<T = any> {
  jsonrpc: '2.0'
  result?: T
  error?: {
    code: number
    message: string
    data?: any
  }
  id: string | number
}

// 创建 axios 实例
const instance: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://127.0.0.1:52281',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
instance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
instance.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// JSON-RPC 调用封装
export async function callRPC<T = any>(
  method: string,
  params?: any
): Promise<T> {
  const request: JSONRPCRequest = {
    jsonrpc: '2.0',
    method,
    params: params || {},
    id: Date.now(),
  }

  try {
    const response = await instance.post<JSONRPCResponse<T>>('/api/rpc', request)
    const data = response.data

    if (data.error) {
      throw new Error(data.error.message)
    }

    if (data.result === undefined) {
      throw new Error('No result in response')
    }

    return data.result
  } catch (error: any) {
    throw new Error(error.message || 'Request failed')
  }
}

export default instance
