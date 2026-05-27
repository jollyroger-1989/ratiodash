import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import './assets/main.css'

import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faHouse,
  faChartBar,
  faEnvelope,
  faBell,
  faGear,
  faRightFromBracket,
  faRotateRight,
  faPen,
  faXmark,
  faPlay,
  faPlus,
  faSpinner,
  faBars,
} from '@fortawesome/free-solid-svg-icons'

library.add(faHouse, faChartBar, faEnvelope, faBell, faGear, faRightFromBracket, faRotateRight, faPen, faXmark, faPlay, faPlus, faSpinner, faBars)

const app = createApp(App)
app.component('font-awesome-icon', FontAwesomeIcon)

app.use(createPinia())
app.use(router)
app.use(i18n)

router.isReady().then(() => app.mount('#app'))
