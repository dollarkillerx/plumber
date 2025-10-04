import { callRPC } from './request'

// 登录参数
export interface LoginParams {
  username: string
  password: string
}

// 登录响应
export interface LoginResponse {
  token: string
  username: string
  user_id: string
}

// 用户登录
export function login(params: LoginParams) {
  return callRPC<LoginResponse>('plumber.user.login', params)
}
