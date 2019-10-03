import Vue from 'vue'
import Router from 'vue-router'
import Home from './views/Home.vue'
import Grid from './views/Grid.vue'

Vue.use(Router)

export default new Router({
  mode: 'history',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home
    },
    {
      path: '/tests/:id',
      name: 'testsid',
      component: Home,
      props: true,
    },
    {
      path: '/tests',
      name: 'tests',
      component: Home
    },
    {
      path: '/tests/:id/logs',
      name: 'testlogs',
      component: Home,
      props: true,
    },
    {
      path: '/grids',
      name: 'grids',
      component: Grid
    },
    {
      path: '/grids/:id',
      name: 'gridsid',
      component: Grid,
      props: true,
    },
    {
      path: '/grids/:id/logs',
      name: 'gridlogs',
      component: Grid,
      props: true,
    },
  ]
})
