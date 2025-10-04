import { callRPC } from './request'

// 任务信息
export interface Task {
  id: string
  name: string
  description: string
  config: string
  status: 'pending' | 'running' | 'success' | 'failed'
  created_at: string
  updated_at: string
}

// 步骤执行信息
export interface StepExecution {
  id: string
  execution_id: string
  step_index: number
  agent_id: string
  path: string
  command: string
  status: 'pending' | 'running' | 'success' | 'failed'
  exit_code?: number
  output?: string
  start_time?: string
  end_time?: string
  created_at: string
  updated_at: string
}

// 任务执行信息
export interface TaskExecution {
  id: string
  task_id: string
  status: 'pending' | 'running' | 'success' | 'failed'
  start_time?: string
  end_time?: string
  created_at: string
  updated_at: string
  steps?: StepExecution[]
}

// 列表响应
export interface ListTasksResponse {
  tasks: Task[]
}

// 创建任务参数
export interface CreateTaskParams {
  name: string
  description: string
  config: string
}

// 创建任务响应
export interface CreateTaskResponse {
  task_id: string
  status: string
}

// 运行任务参数
export interface RunTaskParams {
  task_id: string
}

// 运行任务响应
export interface RunTaskResponse {
  status: string
  message: string
}

// 获取执行记录参数
export interface GetExecutionParams {
  execution_id: string
}

// 获取执行记录响应
export interface GetExecutionResponse {
  execution: TaskExecution
}

// 获取任务列表
export function listTasks() {
  return callRPC<ListTasksResponse>('plumber.task.list')
}

// 创建任务
export function createTask(params: CreateTaskParams) {
  return callRPC<CreateTaskResponse>('plumber.task.create', params)
}

// 运行任务
export function runTask(params: RunTaskParams) {
  return callRPC<RunTaskResponse>('plumber.task.run', params)
}

// 获取执行记录
export function getExecution(params: GetExecutionParams) {
  return callRPC<GetExecutionResponse>('plumber.execution.get', params)
}
