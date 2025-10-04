<template>
  <div>
    <div class="mb-6">
      <h2 class="text-2xl font-bold text-gray-800">Create Task</h2>
    </div>

    <div class="bg-white shadow sm:rounded-lg">
      <div class="px-4 py-5 sm:p-6">
        <form @submit.prevent="handleSubmit" class="space-y-6">
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

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Description
            </label>
            <input
              v-model="form.description"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Task description"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">
              Configuration (TOML) *
            </label>
            <textarea
              v-model="form.config"
              required
              rows="15"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              placeholder="[[step]]
ServerID = &quot;agent-uuid&quot;
Path     = &quot;/opt/app&quot;
CMD      = &quot;git pull&quot;

[[step]]
ServerID = &quot;agent-uuid&quot;
Path     = &quot;/opt/app&quot;
CMD      = &quot;npm install&quot;"
            />
            <p class="mt-2 text-sm text-gray-500">
              Enter TOML configuration with steps. Each step should have ServerID (agent UUID), Path, and CMD.
            </p>
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
              :disabled="loading"
              class="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {{ loading ? 'Creating...' : 'Create Task' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Example hint -->
    <div class="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-4">
      <h3 class="text-sm font-medium text-blue-800 mb-2">💡 Example Configuration:</h3>
      <pre class="text-xs text-blue-700 font-mono bg-blue-100 p-3 rounded overflow-x-auto">[[step]]
ServerID = "00000000-0000-0000-0000-000000000001"
Path     = "/tmp"
CMD      = "echo 'Step 1: Starting'"

[[step]]
ServerID = "00000000-0000-0000-0000-000000000001"
Path     = "/opt/project"
CMD      = "git pull origin main"</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTaskStore } from '@/stores/task'

const router = useRouter()
const taskStore = useTaskStore()

const form = ref({
  name: '',
  description: '',
  config: '',
})

const loading = ref(false)
const error = ref('')

async function handleSubmit() {
  loading.value = true
  error.value = ''

  try {
    await taskStore.createNewTask({
      name: form.value.name,
      description: form.value.description,
      config: form.value.config,
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
