import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  listTasks,
  createTask,
  runTask,
  getExecution,
  type Task,
  type CreateTaskParams,
  type TaskExecution,
} from '@/api/task'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<Task[]>([])
  const currentExecution = ref<TaskExecution | null>(null)
  const loading = ref(false)
  const error = ref<string>('')

  async function fetchTasks() {
    loading.value = true
    error.value = ''
    try {
      const result = await listTasks()
      tasks.value = result.tasks
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createNewTask(params: CreateTaskParams) {
    loading.value = true
    error.value = ''
    try {
      const result = await createTask(params)
      await fetchTasks() // 刷新列表
      return result
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function executeTask(taskId: string) {
    loading.value = true
    error.value = ''
    try {
      const result = await runTask({ task_id: taskId })
      await fetchTasks() // 刷新列表
      return result
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchExecution(executionId: string) {
    loading.value = true
    error.value = ''
    try {
      const result = await getExecution({ execution_id: executionId })
      currentExecution.value = result.execution
      return result.execution
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  function getTaskById(id: string) {
    return tasks.value.find((task) => task.id === id)
  }

  return {
    tasks,
    currentExecution,
    loading,
    error,
    fetchTasks,
    createNewTask,
    executeTask,
    fetchExecution,
    getTaskById,
  }
})
