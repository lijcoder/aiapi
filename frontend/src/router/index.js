import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import Recharge from '../views/Recharge.vue'
import Models from '../views/Models.vue'
import ApiKeys from '../views/ApiKeys.vue'
import Usage from '../views/Usage.vue'

const routes = [
  { path: '/login', name: 'Login', component: Login },
  {
    path: '/',
    component: Home,
    redirect: '/recharge',
    children: [
      { path: 'recharge', name: 'Recharge', component: Recharge },
      { path: 'models', name: 'Models', component: Models },
      { path: 'apikeys', name: 'ApiKeys', component: ApiKeys },
      { path: 'usage', name: 'Usage', component: Usage }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
