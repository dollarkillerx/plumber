<template>
  <div>
    <div class="mb-6 flex justify-between items-center">
      <h2 class="text-2xl font-bold text-gray-800">Task List</h2>
      <div class="space-x-2">
        <button
          @click="fetchData"
          :disabled="loading"
          class="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 disabled:opacity-50"
        >
          {{ loading ? 'Refreshing...' : 'Refresh' }}
        </button>
        <router-link
          to="/tasks/create"
          class="inline-block bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          Create Task
        </router-link>
      </div>
    </div>

    <div v-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
      Error: {{ error }}
    </div>

    <div class="bg-white shadow overflow-hidden sm:rounded-lg">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Name
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Description
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Created At
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-if="loading && tasks.length === 0">
            <td colspan="5" class="px-6 py-4 text-center text-gray-500">
              Loading...
            </td>
          </tr>
          <tr v-else-if="tasks.length === 0">
            <td colspan="5" class="px-6 py-4 text-center text-gray-500">
              No tasks found. Create your first task!
            </td>
          </tr>
          <tr v-for="task in tasks" :key="task.id" class="hover:bg-gray-50">
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
              {{ task.name }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-500">
              {{ task.description }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  'px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                  getStatusClass(task.status),
                ]"
              >
                {{ task.status }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(task.created_at) }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-3">
              <button
                @click="handleRunTask(task.id)"
                :disabled="task.status === 'running' || loading"
                class="text-blue-600 hover:text-blue-900 disabled:text-gray-400 disabled:cursor-not-allowed"
              >
                Run
              </button>
              <button
                @click="showExecutionHistory(task.id, task.name)"
                class="text-green-600 hover:text-green-900"
              >
                History
              </button>
              <button
                @click="openEditModal(task)"
                class="text-indigo-600 hover:text-indigo-900"
              >
                Edit
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit Task Modal -->
    <div
      v-if="showEditModal"
      class="fixed inset-0 overflow-y-auto h-full w-full z-50 flex items-center justify-center"
      @click.self="closeEditModal"
    >
      <div class="relative mx-auto p-6 border border-gray-300 w-full max-w-4xl shadow-xl rounded-lg bg-white max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-medium leading-6 text-gray-900">
            Edit Task
          </h3>
          <button
            @click="closeEditModal"
            class="text-gray-400 hover:text-gray-600"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form @submit.prevent="handleUpdateTask" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700">Task Name *</label>
            <input
              v-model="editForm.name"
              type="text"
              required
              class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700">Description</label>
            <textarea
              v-model="editForm.description"
              rows="2"
              class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            ></textarea>
          </div>

          <!-- Steps -->
          <div>
            <div class="flex justify-between items-center mb-3">
              <label class="block text-sm font-medium text-gray-700">
                Steps *
              </label>
              <button
                type="button"
                @click="addEditStep"
                class="bg-green-600 text-white px-3 py-1 rounded-md hover:bg-green-700 text-sm"
              >
                + Add Step
              </button>
            </div>

            <div v-if="editSteps.length === 0" class="text-center py-8 text-gray-500 border-2 border-dashed border-gray-300 rounded-md">
              No steps added. Click "Add Step" to create a step.
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="(step, index) in editSteps"
                :key="index"
                class="border border-gray-300 rounded-lg p-3 bg-gray-50"
              >
                <div class="flex justify-between items-center mb-2">
                  <h4 class="font-medium text-gray-700 text-sm">Step {{ index + 1 }}</h4>
                  <div class="flex gap-2">
                    <button
                      v-if="index > 0"
                      type="button"
                      @click="moveEditStepUp(index)"
                      class="text-gray-600 hover:text-blue-600"
                      title="Move Up"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                      </svg>
                    </button>
                    <button
                      v-if="index < editSteps.length - 1"
                      type="button"
                      @click="moveEditStepDown(index)"
                      class="text-gray-600 hover:text-blue-600"
                      title="Move Down"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    <button
                      type="button"
                      @click="removeEditStep(index)"
                      class="text-red-600 hover:text-red-800"
                      title="Remove"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>

                <div class="space-y-2">
                  <div>
                    <label class="block text-xs font-medium text-gray-600 mb-1">Agent *</label>
                    <select
                      v-model="step.serverId"
                      required
                      class="w-full px-2 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">Select an agent</option>
                      <option v-for="agent in agents" :key="agent.id" :value="agent.id">
                        {{ agent.name }} ({{ agent.status }})
                      </option>
                    </select>
                  </div>

                  <div>
                    <label class="block text-xs font-medium text-gray-600 mb-1">Path *</label>
                    <input
                      v-model="step.path"
                      type="text"
                      required
                      class="w-full px-2 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="/opt/app"
                    />
                  </div>

                  <div>
                    <label class="block text-xs font-medium text-gray-600 mb-1">
                      Command *
                      <span class="text-gray-400 font-normal text-xs ml-1">(支持多行)</span>
                    </label>
                    <textarea
                      v-model="step.cmd"
                      rows="4"
                      required
                      @keydown.enter.stop
                      class="w-full px-2 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                      placeholder="git pull origin main&#10;npm install&#10;npm run build"
                    ></textarea>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="editError" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
            {{ editError }}
          </div>

          <div class="flex justify-end space-x-3 mt-5">
            <button
              type="button"
              @click="closeEditModal"
              class="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400"
            >
              Cancel
            </button>
            <button
              type="submit"
              :disabled="updating || editSteps.length === 0"
              class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {{ updating ? 'Updating...' : 'Update Task' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Execution History Modal -->
    <div
      v-if="showHistoryModal"
      class="fixed inset-0 overflow-y-auto h-full w-full z-50 flex items-center justify-center"
      @click.self="closeHistoryModal"
    >
      <div class="relative mx-auto p-6 border border-gray-300 w-full max-w-4xl shadow-xl rounded-lg bg-white max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-medium leading-6 text-gray-900">
            Execution History - {{ selectedTaskName }}
          </h3>
          <button
            @click="closeHistoryModal"
            class="text-gray-400 hover:text-gray-600"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div v-if="historyLoading" class="text-center py-8 text-gray-500">
          Loading execution history...
        </div>

        <div v-else-if="historyError" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          Error: {{ historyError }}
        </div>

        <div v-else-if="executionHistory.length === 0" class="text-center py-8 text-gray-500">
          No execution history found for this task.
        </div>

        <div v-else class="space-y-4">
          <div
            v-for="execution in executionHistory"
            :key="execution.id"
            class="border border-gray-200 rounded-lg p-4"
          >
            <div class="flex justify-between items-start mb-3">
              <div>
                <span
                  :class="[
                    'px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                    getStatusClass(execution.status),
                  ]"
                >
                  {{ execution.status }}
                </span>
                <span class="ml-2 text-sm text-gray-500">
                  ID: {{ execution.id.substring(0, 8) }}
                </span>
              </div>
              <div class="text-right text-sm text-gray-500">
                <div>Started: {{ formatDate(execution.created_at) }}</div>
                <div v-if="execution.end_time">Ended: {{ formatDate(execution.end_time) }}</div>
              </div>
            </div>
            <div v-if="execution.start_time && execution.end_time" class="text-sm text-gray-600 mb-3">
              Duration: {{ calculateDuration(execution.start_time, execution.end_time) }}
            </div>

            <!-- Steps -->
            <div v-if="execution.steps && execution.steps.length > 0" class="mt-3 space-y-3">
              <div class="text-sm font-medium text-gray-700 border-t pt-3">
                Steps ({{ execution.steps.length }}):
              </div>
              <div
                v-for="(step, index) in execution.steps"
                :key="step.id"
                class="bg-gray-50 rounded-md p-3 space-y-2"
              >
                <div class="flex justify-between items-start">
                  <div class="flex items-center space-x-2">
                    <span class="text-xs font-semibold text-gray-500">Step {{ index + 1 }}</span>
                    <span
                      :class="[
                        'px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                        getStatusClass(step.status),
                      ]"
                    >
                      {{ step.status }}
                    </span>
                  </div>
                  <div v-if="step.exit_code !== null" class="text-xs text-gray-500">
                    Exit Code: {{ step.exit_code }}
                  </div>
                </div>

                <div class="text-xs text-gray-600">
                  <div><strong>Path:</strong> {{ step.path }}</div>
                  <div><strong>Command:</strong> {{ step.command }}</div>
                </div>

                <div v-if="step.start_time && step.end_time" class="text-xs text-gray-500">
                  Duration: {{ calculateDuration(step.start_time, step.end_time) }}
                </div>

                <div v-if="step.output" class="mt-2">
                  <div class="text-xs font-medium text-gray-700 mb-1">Output:</div>
                  <pre class="bg-gray-900 text-green-400 text-xs p-3 rounded overflow-x-auto max-h-60 overflow-y-auto">{{ step.output }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useTaskStore } from '@/stores/task'
import { useAgentStore } from '@/stores/agent'
import { listExecutions, updateTask, type TaskExecution } from '@/api/task'

const taskStore = useTaskStore()
const agentStore = useAgentStore()

const tasks = computed(() => taskStore.tasks)
const loading = computed(() => taskStore.loading)
const error = computed(() => taskStore.error)
const agents = computed(() => agentStore.agents)

interface Step {
  serverId: string
  path: string
  cmd: string
}

const showHistoryModal = ref(false)
const historyLoading = ref(false)
const historyError = ref('')
const selectedTaskName = ref('')
const executionHistory = ref<TaskExecution[]>([])

const showEditModal = ref(false)
const editError = ref('')
const updating = ref(false)
const editingTaskId = ref('')
const editForm = ref({
  name: '',
  description: ''
})
const editSteps = ref<Step[]>([])

onMounted(() => {
  fetchData()
  fetchAgents()
})

async function fetchAgents() {
  try {
    await agentStore.fetchAgents()
  } catch (err) {
    console.error('Failed to fetch agents:', err)
  }
}

async function fetchData() {
  try {
    await taskStore.fetchTasks()
  } catch (err) {
    console.error('Failed to fetch tasks:', err)
  }
}

async function handleRunTask(taskId: string) {
  if (!confirm('Are you sure you want to run this task?')) return

  try {
    await taskStore.executeTask(taskId)
    alert('Task execution started!')
  } catch (err: any) {
    alert(`Failed to run task: ${err.message}`)
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString()
}

function getStatusClass(status: string) {
  switch (status) {
    case 'success':
      return 'bg-green-100 text-green-800'
    case 'failed':
      return 'bg-red-100 text-red-800'
    case 'running':
      return 'bg-blue-100 text-blue-800'
    default:
      return 'bg-gray-100 text-gray-800'
  }
}

async function showExecutionHistory(taskId: string, taskName: string) {
  selectedTaskName.value = taskName
  showHistoryModal.value = true
  historyLoading.value = true
  historyError.value = ''
  executionHistory.value = []

  try {
    const response = await listExecutions({ task_id: taskId })
    executionHistory.value = response.executions || []
  } catch (err: any) {
    historyError.value = err.message || 'Failed to load execution history'
  } finally {
    historyLoading.value = false
  }
}

function closeHistoryModal() {
  showHistoryModal.value = false
  selectedTaskName.value = ''
  executionHistory.value = []
  historyError.value = ''
}

function calculateDuration(start: string, end: string) {
  const startTime = new Date(start).getTime()
  const endTime = new Date(end).getTime()
  const duration = Math.floor((endTime - startTime) / 1000)

  if (duration < 60) {
    return `${duration}s`
  } else if (duration < 3600) {
    const minutes = Math.floor(duration / 60)
    const seconds = duration % 60
    return `${minutes}m ${seconds}s`
  } else {
    const hours = Math.floor(duration / 3600)
    const minutes = Math.floor((duration % 3600) / 60)
    return `${hours}h ${minutes}m`
  }
}

function parseTOMLToSteps(config: string): Step[] {
  const steps: Step[] = []
  const stepBlocks = config.split('[[step]]').filter(s => s.trim())

  for (const block of stepBlocks) {
    const serverIdMatch = block.match(/ServerID\s*=\s*"([^"]+)"/)
    const pathMatch = block.match(/Path\s*=\s*"([^"]+)"/)

    // 支持单行和多行命令
    let cmdMatch = block.match(/CMD\s*=\s*"""([\s\S]*?)"""/)
    if (!cmdMatch) {
      cmdMatch = block.match(/CMD\s*=\s*"([^"]+)"/)
    }

    if (serverIdMatch && pathMatch && cmdMatch) {
      steps.push({
        serverId: serverIdMatch[1],
        path: pathMatch[1],
        cmd: cmdMatch[1].trim()
      })
    }
  }

  return steps
}

function generateTOMLFromSteps(steps: Step[]): string {
  let toml = ''
  for (const step of steps) {
    toml += `[[step]]\n`
    toml += `ServerID = "${step.serverId}"\n`
    toml += `Path     = "${step.path}"\n`

    // 处理多行命令 - 如果命令包含换行符，使用三引号语法
    if (step.cmd.includes('\n')) {
      toml += `CMD      = """\n${step.cmd}\n"""\n\n`
    } else {
      toml += `CMD      = "${step.cmd}"\n\n`
    }
  }
  return toml.trim()
}

function openEditModal(task: any) {
  editingTaskId.value = task.id
  editForm.value = {
    name: task.name,
    description: task.description
  }
  editSteps.value = parseTOMLToSteps(task.config)
  showEditModal.value = true
  editError.value = ''
}

function closeEditModal() {
  showEditModal.value = false
  editingTaskId.value = ''
  editError.value = ''
  editForm.value = {
    name: '',
    description: ''
  }
  editSteps.value = []
}

function addEditStep() {
  editSteps.value.push({
    serverId: '',
    path: '',
    cmd: ''
  })
}

function removeEditStep(index: number) {
  editSteps.value.splice(index, 1)
}

function moveEditStepUp(index: number) {
  if (index > 0) {
    const temp = editSteps.value[index]
    editSteps.value[index] = editSteps.value[index - 1]
    editSteps.value[index - 1] = temp
  }
}

function moveEditStepDown(index: number) {
  if (index < editSteps.value.length - 1) {
    const temp = editSteps.value[index]
    editSteps.value[index] = editSteps.value[index + 1]
    editSteps.value[index + 1] = temp
  }
}

async function handleUpdateTask() {
  updating.value = true
  editError.value = ''

  try {
    const config = generateTOMLFromSteps(editSteps.value)
    await updateTask({
      task_id: editingTaskId.value,
      name: editForm.value.name,
      description: editForm.value.description,
      config: config
    })
    closeEditModal()
    await fetchData()
  } catch (err: any) {
    editError.value = err.message || 'Failed to update task'
  } finally {
    updating.value = false
  }
}
</script>
