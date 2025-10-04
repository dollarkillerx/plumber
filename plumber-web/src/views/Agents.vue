<template>
  <div>
    <div class="mb-6 flex justify-between items-center">
      <h2 class="text-2xl font-bold text-gray-800">Agent Management</h2>
      <div class="space-x-2">
        <button
          @click="showConfigModal = true"
          class="bg-purple-600 text-white px-4 py-2 rounded-md hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500"
        >
          Deploy Config
        </button>
        <button
          @click="showCreateModal = true"
          class="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500"
        >
          Create Agent
        </button>
        <button
          @click="fetchData"
          :disabled="loading"
          class="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {{ loading ? 'Refreshing...' : 'Refresh' }}
        </button>
      </div>
    </div>

    <div v-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
      Error: {{ error }}
    </div>

    <div class="bg-white shadow overflow-x-auto sm:rounded-lg">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Name / ID
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              SSH Info
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Hostname/IP
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Last Heartbeat
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-if="loading && agents.length === 0">
            <td colspan="6" class="px-6 py-4 text-center text-gray-500">
              Loading...
            </td>
          </tr>
          <tr v-else-if="agents.length === 0">
            <td colspan="6" class="px-6 py-4 text-center text-gray-500">
              No agents found. Click "Create Agent" to add one.
            </td>
          </tr>
          <tr v-for="agent in agents" :key="agent.id" class="hover:bg-gray-50">
            <td class="px-6 py-4 text-sm">
              <div class="font-medium text-gray-900 whitespace-nowrap">{{ agent.name }}</div>
              <div class="flex items-center gap-2 mt-1">
                <span class="text-xs text-gray-400 font-mono truncate max-w-[200px]" :title="agent.id">{{ agent.id }}</span>
                <button
                  @click="copyToClipboard(agent.id)"
                  class="text-gray-400 hover:text-blue-600 transition-colors flex-shrink-0"
                  title="Copy UUID"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                </button>
              </div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              <span v-if="agent.ssh_host">
                {{ agent.ssh_user }}@{{ agent.ssh_host }}:{{ agent.ssh_port || 22 }}
              </span>
              <span v-else class="text-gray-400">-</span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              <div v-if="agent.hostname || agent.ip">
                <div v-if="agent.hostname">{{ agent.hostname }}</div>
                <div v-if="agent.ip" class="text-xs text-gray-400">{{ agent.ip }}</div>
              </div>
              <span v-else class="text-gray-400">Not connected</span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  'px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                  agent.status === 'online'
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800',
                ]"
              >
                {{ agent.status }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ agent.last_heartbeat ? formatDate(agent.last_heartbeat) : '-' }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-500">
              <div class="flex flex-wrap gap-2">
                <button
                  v-if="agent.ssh_host"
                  @click="handleDeploy(agent)"
                  :disabled="deploying === agent.id"
                  class="text-purple-600 hover:text-purple-900 whitespace-nowrap disabled:opacity-50"
                >
                  {{ deploying === agent.id ? 'Deploying...' : 'Deploy' }}
                </button>
                <button
                  @click="openEditModal(agent)"
                  class="text-indigo-600 hover:text-indigo-900 whitespace-nowrap"
                >
                  Edit
                </button>
                <button
                  @click="downloadConfig(agent.id)"
                  class="text-blue-600 hover:text-blue-900 whitespace-nowrap"
                >
                  Download Config
                </button>
                <button
                  v-if="agent.ssh_host"
                  @click="openWebSSH(agent)"
                  class="text-green-600 hover:text-green-900 whitespace-nowrap"
                >
                  WebSSH
                </button>
                <button
                  @click="handleDelete(agent.id, agent.name)"
                  class="text-red-600 hover:text-red-900 whitespace-nowrap"
                >
                  Delete
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Deploy Config Modal -->
    <div
      v-if="showConfigModal"
      class="fixed inset-0 overflow-y-auto h-full w-full z-50 flex items-center justify-center"
      @click.self="showConfigModal = false"
    >
      <div class="relative mx-auto p-6 border border-gray-300 w-full max-w-2xl shadow-xl rounded-lg bg-white">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-medium leading-6 text-gray-900">
            Deploy Configuration
          </h3>
          <button
            @click="showConfigModal = false"
            class="text-gray-400 hover:text-gray-600"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">Install Script URL</label>
            <input
              v-model="deployConfig.scriptUrl"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="https://raw.githubusercontent.com/.../install_agent.sh"
            />
            <p class="mt-1 text-xs text-gray-500">脚本将通过 SSH 下载并执行</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">Install Directory</label>
            <input
              v-model="deployConfig.installDir"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="/opt/plumber_agent"
            />
            <p class="mt-1 text-xs text-gray-500">Agent 安装目录，部署前会被删除</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">Server Address</label>
            <input
              v-model="deployConfig.serverAddr"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="http://your-server:52281"
            />
            <p class="mt-1 text-xs text-gray-500">Plumber Server 地址，写入 Agent 配置文件</p>
          </div>
          <div class="flex justify-end space-x-3 pt-4">
            <button
              @click="showConfigModal = false"
              class="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400"
            >
              Close
            </button>
            <button
              @click="saveDeployConfig"
              class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >
              Save
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Agent Modal -->
    <div
      v-if="showCreateModal || showEditModal"
      class="fixed inset-0 overflow-y-auto h-full w-full z-50 flex items-center justify-center"
      @click.self="closeModals"
    >
      <div class="relative mx-auto p-6 border border-gray-300 w-full max-w-md shadow-xl rounded-lg bg-white">
        <div>
          <h3 class="text-lg font-medium leading-6 text-gray-900 mb-4">
            {{ showEditModal ? 'Edit Agent' : 'Create New Agent' }}
          </h3>
          <form @submit.prevent="showEditModal ? handleUpdateAgent() : handleCreateAgent()" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700">Name *</label>
              <input
                v-model="createForm.name"
                type="text"
                required
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="My Server"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700">SSH Host</label>
              <input
                v-model="createForm.ssh_host"
                type="text"
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="192.168.1.100"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700">SSH Port</label>
              <input
                v-model.number="createForm.ssh_port"
                type="number"
                min="1"
                max="65535"
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="22 (default)"
              />
              <p class="mt-1 text-sm text-gray-500">Default: 22</p>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700">SSH User</label>
              <input
                v-model="createForm.ssh_user"
                type="text"
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="root"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700">Auth Type</label>
              <select
                v-model="createForm.ssh_auth_type"
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="none">None</option>
                <option value="password">Password</option>
                <option value="key">SSH Key</option>
              </select>
            </div>
            <div v-if="createForm.ssh_auth_type === 'password'">
              <label class="block text-sm font-medium text-gray-700">SSH Password</label>
              <input
                v-model="createForm.ssh_password"
                type="password"
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="Password"
              />
            </div>
            <div v-if="createForm.ssh_auth_type === 'key'">
              <label class="block text-sm font-medium text-gray-700">SSH Private Key</label>
              <textarea
                v-model="createForm.ssh_private_key"
                rows="4"
                class="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 font-mono text-xs"
                placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
              ></textarea>
            </div>
            <div v-if="createError" class="text-red-600 text-sm">
              {{ createError }}
            </div>
            <div class="flex justify-end space-x-3 mt-5">
              <button
                type="button"
                @click="closeModals"
                class="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400"
              >
                Cancel
              </button>
              <button
                type="submit"
                :disabled="creating || updating"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
              >
                {{ showEditModal ? (updating ? 'Updating...' : 'Update') : (creating ? 'Creating...' : 'Create') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAgentStore } from '@/stores/agent'
import { createAgent, getAgentConfig, updateAgent, deleteAgent } from '@/api/agent'
import type { CreateAgentParams, UpdateAgentParams, Agent } from '@/api/agent'

const router = useRouter()
const agentStore = useAgentStore()

const agents = computed(() => agentStore.agents)
const loading = computed(() => agentStore.loading)
const error = computed(() => agentStore.error)

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showConfigModal = ref(false)
const creating = ref(false)
const updating = ref(false)
const deploying = ref('')
const createError = ref('')
const editingAgentId = ref('')
const createForm = ref<CreateAgentParams>({
  name: '',
  ssh_host: '',
  ssh_port: 22,
  ssh_user: '',
  ssh_auth_type: 'none',
  ssh_password: '',
  ssh_private_key: '',
})

// 部署配置 - 从 localStorage 加载
const deployConfig = ref({
  scriptUrl: localStorage.getItem('plumber_deploy_script_url') || 'https://raw.githubusercontent.com/dollarkillerx/plumber/refs/heads/main/scripts/install_agent.sh',
  installDir: localStorage.getItem('plumber_deploy_install_dir') || '/opt/plumber_agent',
  serverAddr: localStorage.getItem('plumber_deploy_server_addr') || 'http://localhost:52281',
})

onMounted(() => {
  fetchData()
})

async function fetchData() {
  try {
    await agentStore.fetchAgents()
  } catch (err) {
    console.error('Failed to fetch agents:', err)
  }
}

function closeModals() {
  showCreateModal.value = false
  showEditModal.value = false
  createError.value = ''
  editingAgentId.value = ''
  createForm.value = {
    name: '',
    ssh_host: '',
    ssh_port: 22,
    ssh_user: '',
    ssh_auth_type: 'none',
    ssh_password: '',
    ssh_private_key: '',
  }
}

function openEditModal(agent: Agent) {
  editingAgentId.value = agent.id
  createForm.value = {
    name: agent.name,
    ssh_host: agent.ssh_host || '',
    ssh_port: agent.ssh_port || 22,
    ssh_user: agent.ssh_user || '',
    ssh_auth_type: agent.ssh_auth_type || 'none',
    ssh_password: agent.ssh_password || '',
    ssh_private_key: agent.ssh_private_key || '',
  }
  showEditModal.value = true
}

async function handleCreateAgent() {
  creating.value = true
  createError.value = ''

  try {
    await createAgent(createForm.value)
    closeModals()
    await fetchData()
  } catch (err: any) {
    createError.value = err.message || 'Failed to create agent'
  } finally {
    creating.value = false
  }
}

async function handleUpdateAgent() {
  updating.value = true
  createError.value = ''

  try {
    const params: UpdateAgentParams = {
      agent_id: editingAgentId.value,
      ...createForm.value,
    }
    await updateAgent(params)
    closeModals()
    await fetchData()
  } catch (err: any) {
    createError.value = err.message || 'Failed to update agent'
  } finally {
    updating.value = false
  }
}

async function handleDelete(agentId: string, agentName: string) {
  if (!confirm(`Are you sure you want to delete agent "${agentName}"?`)) {
    return
  }

  try {
    await deleteAgent(agentId)
    await fetchData()
  } catch (err: any) {
    alert('Failed to delete agent: ' + (err.message || 'Unknown error'))
  }
}

function openWebSSH(agent: Agent) {
  // 跳转到 WebSSH 页面
  router.push({
    name: 'webssh',
    query: { agent_id: agent.id }
  })
}

async function downloadConfig(agentId: string) {
  try {
    const response = await getAgentConfig(agentId)

    // 创建下载
    const blob = new Blob([response.config], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = response.filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (err: any) {
    alert('Failed to download config: ' + (err.message || 'Unknown error'))
  }
}

function formatDate(date: string) {
  return new Date(date).toLocaleString()
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    // 可以添加一个临时提示
    alert('UUID copied to clipboard!')
  }).catch(err => {
    console.error('Failed to copy:', err)
  })
}

function saveDeployConfig() {
  localStorage.setItem('plumber_deploy_script_url', deployConfig.value.scriptUrl)
  localStorage.setItem('plumber_deploy_install_dir', deployConfig.value.installDir)
  localStorage.setItem('plumber_deploy_server_addr', deployConfig.value.serverAddr)
  showConfigModal.value = false
  alert('Deploy configuration saved!')
}

async function handleDeploy(agent: Agent) {
  if (!confirm(`Deploy agent to ${agent.name} (${agent.ssh_host})?\\n\\nThis will:\\n1. Delete ${deployConfig.value.installDir}\\n2. Write agent config\\n3. Run install script\\n\\nContinue?`)) {
    return
  }

  deploying.value = agent.id

  try {
    // 调用部署 API
    const response = await fetch('/api/rpc', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        method: 'plumber.agent.deploy',
        params: {
          agent_id: agent.id,
          script_url: deployConfig.value.scriptUrl,
          install_dir: deployConfig.value.installDir,
          server_addr: deployConfig.value.serverAddr,
        },
        id: Date.now().toString(),
      }),
    })

    const result = await response.json()

    if (result.error) {
      throw new Error(result.error.message || 'Deploy failed')
    }

    alert(`Deploy successful!\\n\\nOutput:\\n${result.result?.output || 'No output'}`)
    await fetchData()
  } catch (err: any) {
    alert('Deploy failed: ' + (err.message || 'Unknown error'))
  } finally {
    deploying.value = ''
  }
}
</script>
