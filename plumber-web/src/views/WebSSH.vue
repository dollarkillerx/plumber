<template>
  <div class="h-screen flex flex-col bg-black">
    <div class="bg-gray-800 px-4 py-2 flex justify-between items-center">
      <div class="text-white text-sm">
        <span v-if="agentName">{{ agentName }}</span>
        <span v-else>WebSSH Terminal</span>
      </div>
      <button
        @click="goBack"
        class="text-gray-400 hover:text-white text-sm"
      >
        ← Back
      </button>
    </div>
    <div ref="terminalContainer" class="flex-1"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const route = useRoute()
const router = useRouter()

const terminalContainer = ref<HTMLElement>()
const agentName = ref('')

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null

onMounted(() => {
  const agentId = route.query.agent_id as string
  if (!agentId) {
    alert('Agent ID is required')
    router.push({ name: 'agents' })
    return
  }

  initTerminal(agentId)
})

onBeforeUnmount(() => {
  cleanup()
})

function initTerminal(agentId: string) {
  if (!terminalContainer.value) return

  // 创建终端
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#000000',
      foreground: '#ffffff',
    },
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)

  terminal.open(terminalContainer.value)
  fitAddon.fit()

  // 监听窗口大小变化
  window.addEventListener('resize', handleResize)

  // 连接 WebSocket
  connectWebSocket(agentId)

  // 监听终端输入
  terminal.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'data',
        data: data,
      }))
    }
  })
}

function connectWebSocket(agentId: string) {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsHost = import.meta.env.VITE_API_URL?.replace(/^https?:\/\//, '') || window.location.host
  const wsUrl = `${wsProtocol}//${wsHost}/api/webssh?agent_id=${agentId}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket connected')
    if (terminal && fitAddon) {
      // 发送终端大小
      ws?.send(JSON.stringify({
        type: 'resize',
        rows: terminal.rows,
        cols: terminal.cols,
      }))
    }
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'data') {
        terminal?.write(msg.data)
      } else if (msg.type === 'error') {
        terminal?.write(`\r\n\x1b[31mError: ${msg.data}\x1b[0m\r\n`)
      }
    } catch (err) {
      console.error('Failed to parse message:', err)
    }
  }

  ws.onerror = (error) => {
    console.error('WebSocket error:', error)
    terminal?.write('\r\n\x1b[31mWebSocket connection error\x1b[0m\r\n')
  }

  ws.onclose = () => {
    console.log('WebSocket closed')
    terminal?.write('\r\n\x1b[33mConnection closed\x1b[0m\r\n')
  }
}

function handleResize() {
  if (fitAddon && terminal && ws && ws.readyState === WebSocket.OPEN) {
    fitAddon.fit()
    ws.send(JSON.stringify({
      type: 'resize',
      rows: terminal.rows,
      cols: terminal.cols,
    }))
  }
}

function cleanup() {
  window.removeEventListener('resize', handleResize)
  if (ws) {
    ws.close()
    ws = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
}

function goBack() {
  router.push({ name: 'agents' })
}
</script>
