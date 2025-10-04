import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: '/agents',
        },
        {
          path: 'agents',
          name: 'agents',
          component: () => import('@/views/Agents.vue'),
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: () => import('@/views/Tasks.vue'),
        },
        {
          path: 'tasks/create',
          name: 'task-create',
          component: () => import('@/views/TaskCreate.vue'),
        },
        {
          path: 'tasks/:id/execution/:executionId',
          name: 'task-execution',
          component: () => import('@/views/TaskExecution.vue'),
        },
      ],
    },
    {
      path: '/webssh',
      name: 'webssh',
      component: () => import('@/views/WebSSH.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    next({ name: 'login' })
  } else if (to.name === 'login' && authStore.isLoggedIn) {
    next({ name: 'agents' })
  } else {
    next()
  }
})

export default router
