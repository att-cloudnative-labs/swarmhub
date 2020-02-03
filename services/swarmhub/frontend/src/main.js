import Vue from 'vue'
import App from './App.vue'
import router from './router'
import './../node_modules/bulma/css/bulma.css';

import { library } from '@fortawesome/fontawesome-svg-core'
import { faCoffee, faEdit, faTrash, faDownload, faUpload, faPlusCircle, faMinusCircle, faSave } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

library.add(faCoffee, faEdit, faTrash, faDownload, faUpload, faPlusCircle, faMinusCircle, faSave)

Vue.use(require('vue-moment'));

Vue.component('font-awesome-icon', FontAwesomeIcon)

Vue.config.productionTip = false

new Vue({
  router,
  render: h => h(App)
}).$mount('#app')
