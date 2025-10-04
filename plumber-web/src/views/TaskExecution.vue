<template>
  <div>
    <div class="mb-6 flex justify-between items-center">
      <h2 class="text-2xl font-bold text-gray-800">Task Execution Details</h2>
      <div class="space-x-2">
        <button
          @click="fetchData"
          :disabled="loading"
          class="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 disabled:opacity-50"
        >
          {{ loading ? 'Refreshing...' : 'Refresh' }}
        </button>
        <router-link
          to="/tasks"
          class="inline-block bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          Back to Tasks
        </router-link>
      </div>
    </div>

    <div v-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
      Error: {{ error }}
    </div>

    <div v-if="execution" class="space-y-6">
      <!-- Execution info -->
      <div class="bg-white shadow rounded-lg p-6">
        <h3 class="text-lg font-medium text-gray-900 mb-4">Execution Info</h3>
        <dl class="grid grid-cols-2 gap-4">
          <div>
            <dt class="text-sm font-medium text-gray-500">Execution ID</dt>
            <dd class="mt-1 text-sm text-gray-900 font-mono">{{ execution.id }}</dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-gray-500">Status</dt>
            <dd class="mt-1">
              <span
                :class="[
                  'px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                  getStatusClass(execution.status),
                ]"
              >
                {{ execution.status }}
              </span>
            </dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-gray-500">Start Time</dt>
            <dd class="mt-1 text-sm text-gray-900">
              {{ execution.start_time ? formatDate(execution.start_time) : 'N/A' }}
            </dd>
          </div>
          <div>
            <dt class="text-sm font-medium text-gray-500">End Time</dt>
            <dd class="mt-1 text-sm text-gray-900">
              {{ execution.end_time ? formatDate(execution.end_time) : 'N/A' }}
            </dd>
          </div>
        </dl>
      </div>

      <!-- Steps -->
      <div class="bg-white shadow rounded-lg p-6">
        <h3 class="text-lg font-medium text-gray-900 mb-4">Execution Steps</h3>
        <div v-if="!execution.steps || execution.steps.length === 0" class="text-gray-500">
          No steps found
        </div>
        <div v-else class="space-y-4">
          <div
            v-for="step in execution.steps"
            :key="step.id"
            class="border border-gray-200 rounded-lg p-4"
          >
            <div class="flex justify-between items-start mb-2">
              <div class="flex items-center">
                <span class="text-sm font-medium text-gray-700">
                  Step {{ step.step_index + 1 }}
                </span>
                <span
                  :class="[
                    'ml-3 px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                    getStatusClass(step.status),
                  ]"
                >
                  {{ step.status }}
                </span>
              </div>
              <span v-if="step.exit_code !== undefined" class="text-sm text-gray-500">
                Exit Code: {{ step.exit_code }}
              </span>
            </div>

            <div class="space-y-2 text-sm">
              <div>
                <span class="font-medium text-gray-700">Agent:</span>
                <span class="ml-2 text-gray-600 font-mono">{{ step.agent_id }}</span>
              </div>
              <div>
                <span class="font-medium text-gray-700">Path:</span>
                <span class="ml-2 text-gray-600">{{ step.path }}</span>
              </div>
              <div>
                <span class="font-medium text-gray-700">Command:</span>
                <span class="ml-2 text-gray-600 font-mono">{{ step.command }}</span>
              </div>

              <div v-if="step.output" class="mt-3">
                <span class="font-medium text-gray-700 block mb-1">Output:</span>
                <pre class="bg-gray-900 text-green-400 p-3 rounded overflow-x-auto text-xs font-mono">{{ step.output }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="loading" class="text-center py-12 text-gray-500">
      Loading execution details...
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useTaskStore } from '@/stores/task'

const route = useRoute()
const taskStore = useTaskStore()

const loading = ref(false)
const error = ref('')

const execution = computed(() => taskStore.currentExecution)
const executionId = computed(() => route.params.executionId as string)

onMounted(() => {
  fetchData()
})

async function fetchData() {
  loading.value = true
  error.value = ''

  try {
    await taskStore.fetchExecution(executionId.value)
  } catch (err: any) {
    error.value = err.message
  } finally {
    loading.value = false
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
</script>
