import { callRPC } from './request'

// Agent 信息
export interface Agent {
  id: string
  name: string
  ssh_host?: string
  ssh_port?: number
  ssh_user?: string
  ssh_auth_type?: 'none' | 'password' | 'key'
  ssh_password?: string
  ssh_private_key?: string
  hostname?: string
  ip?: string
  status: 'online' | 'offline'
  last_heartbeat?: string
  created_at: string
  updated_at: string
}

// 列表响应
export interface ListAgentsResponse {
  agents: Agent[]
}

// 创建 Agent 参数
export interface CreateAgentParams {
  name: string
  ssh_host?: string
  ssh_port?: number
  ssh_user?: string
  ssh_auth_type?: 'none' | 'password' | 'key'
  ssh_password?: string
  ssh_private_key?: string
}

// 创建 Agent 响应
export interface CreateAgentResponse {
  agent_id: string
  name: string
  status: string
}

// 获取配置响应
export interface GetAgentConfigResponse {
  config: string
  filename: string
}

// 获取 Agent 列表
export function listAgents() {
  return callRPC<ListAgentsResponse>('plumber.agent.list')
}

// 创建 Agent
export function createAgent(params: CreateAgentParams) {
  return callRPC<CreateAgentResponse>('plumber.agent.create', params)
}

// 获取 Agent 配置文件
export function getAgentConfig(agentId: string) {
  return callRPC<GetAgentConfigResponse>('plumber.agent.getConfig', { agent_id: agentId })
}

// 更新 Agent
export interface UpdateAgentParams {
  agent_id: string
  name: string
  ssh_host?: string
  ssh_port?: number
  ssh_user?: string
  ssh_auth_type?: 'none' | 'password' | 'key'
  ssh_password?: string
  ssh_private_key?: string
}

export function updateAgent(params: UpdateAgentParams) {
  return callRPC('plumber.agent.update', params)
}

// 删除 Agent
export function deleteAgent(agentId: string) {
  return callRPC('plumber.agent.delete', { agent_id: agentId })
}
