import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import Recharge from '../views/Recharge.vue'

const routes = [
  { path: '/login', name: 'Login', component: Login },
  {
    path: '/',
    component: Home,
    children: [
      { path: '', name: 'Recharge', component: Recharge }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
