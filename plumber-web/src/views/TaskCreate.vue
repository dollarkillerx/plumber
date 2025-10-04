<template>
  <div>
    <div class="mb-6">
      <h2 class="text-2xl font-bold text-gray-800">Create Task</h2>
    </div>

    <div class="bg-white shadow sm:rounded-lg">
      <div class="px-4 py-5 sm:p-6">
        <form @submit.prevent="handleSubmit" class="space-y-6">
          <!-- Task Name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Task Name *
            </label>
            <input
              v-model="form.name"
              type="text"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="e.g., Deploy Production"
            />
          </div>

          <!-- Description -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Description
            </label>
            <textarea
              v-model="form.description"
              rows="2"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Task description"
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
                @click="addStep"
                class="bg-green-600 text-white px-3 py-1 rounded-md hover:bg-green-700 text-sm"
              >
                + Add Step
              </button>
            </div>

            <div v-if="steps.length === 0" class="text-center py-8 text-gray-500 border-2 border-dashed border-gray-300 rounded-md">
              No steps added. Click "Add Step" to create your first step.
            </div>

            <div v-else class="space-y-4">
              <div
                v-for="(step, index) in steps"
                :key="index"
                class="border border-gray-300 rounded-lg p-4 bg-gray-50"
              >
                <div class="flex justify-between items-center mb-3">
                  <h3 class="font-medium text-gray-700">Step {{ index + 1 }}</h3>
                  <div class="flex gap-2">
                    <button
                      v-if="index > 0"
                      type="button"
                      @click="moveStepUp(index)"
                      class="text-gray-600 hover:text-blue-600"
                      title="Move Up"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                      </svg>
                    </button>
                    <button
                      v-if="index < steps.length - 1"
                      type="button"
                      @click="moveStepDown(index)"
                      class="text-gray-600 hover:text-blue-600"
                      title="Move Down"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    <button
                      type="button"
                      @click="removeStep(index)"
                      class="text-red-600 hover:text-red-800"
                      title="Remove"
                    >
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>

                <div class="space-y-3">
                  <!-- Agent Selection -->
                  <div>
                    <label class="block text-sm font-medium text-gray-600 mb-1">Agent *</label>
                    <select
                      v-model="step.serverId"
                      required
                      class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">Select an agent</option>
                      <option v-for="agent in agents" :key="agent.id" :value="agent.id">
                        {{ agent.name }} ({{ agent.status }})
                      </option>
                    </select>
                  </div>

                  <!-- Path -->
                  <div>
                    <label class="block text-sm font-medium text-gray-600 mb-1">Path *</label>
                    <input
                      v-model="step.path"
                      type="text"
                      required
                      class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="/opt/app"
                    />
                  </div>

                  <!-- Command -->
                  <div>
                    <label class="block text-sm font-medium text-gray-600 mb-1">Command *</label>
                    <input
                      v-model="step.cmd"
                      type="text"
                      required
                      class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="git pull origin main"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="error" class="text-red-600 text-sm">
            {{ error }}
          </div>

          <div class="flex justify-end space-x-3">
            <router-link
              to="/tasks"
              class="bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-500"
            >
              Cancel
            </router-link>
            <button
              type="submit"
              :disabled="loading || steps.length === 0"
              class="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {{ loading ? 'Creating...' : 'Create Task' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useTaskStore } from '@/stores/task'
import { useAgentStore } from '@/stores/agent'

const router = useRouter()
const taskStore = useTaskStore()
const agentStore = useAgentStore()

const agents = computed(() => agentStore.agents)

interface Step {
  serverId: string
  path: string
  cmd: string
}

const form = ref({
  name: '',
  description: '',
})

const steps = ref<Step[]>([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    await agentStore.fetchAgents()
  } catch (err) {
    console.error('Failed to fetch agents:', err)
  }
})

function addStep() {
  steps.value.push({
    serverId: '',
    path: '',
    cmd: ''
  })
}

function removeStep(index: number) {
  steps.value.splice(index, 1)
}

function moveStepUp(index: number) {
  if (index > 0) {
    const temp = steps.value[index]
    steps.value[index] = steps.value[index - 1]
    steps.value[index - 1] = temp
  }
}

function moveStepDown(index: number) {
  if (index < steps.value.length - 1) {
    const temp = steps.value[index]
    steps.value[index] = steps.value[index + 1]
    steps.value[index + 1] = temp
  }
}

function generateTOML(): string {
  let toml = ''
  for (const step of steps.value) {
    toml += `[[step]]\n`
    toml += `ServerID = "${step.serverId}"\n`
    toml += `Path     = "${step.path}"\n`
    toml += `CMD      = "${step.cmd}"\n\n`
  }
  return toml.trim()
}

async function handleSubmit() {
  loading.value = true
  error.value = ''

  try {
    const config = generateTOML()
    await taskStore.createNewTask({
      name: form.value.name,
      description: form.value.description,
      config: config,
    })
    alert('Task created successfully!')
    router.push({ name: 'tasks' })
  } catch (err: any) {
    error.value = err.message || 'Failed to create task'
  } finally {
    loading.value = false
  }
}
</script>
