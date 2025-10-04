import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listAgents, type Agent } from '@/api/agent'

export const useAgentStore = defineStore('agent', () => {
  const agents = ref<Agent[]>([])
  const loading = ref(false)
  const error = ref<string>('')

  async function fetchAgents() {
    loading.value = true
    error.value = ''
    try {
      const result = await listAgents()
      agents.value = result.agents
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  function getAgentById(id: string) {
    return agents.value.find((agent) => agent.id === id)
  }

  return {
    agents,
    loading,
    error,
    fetchAgents,
    getAgentById,
  }
})
