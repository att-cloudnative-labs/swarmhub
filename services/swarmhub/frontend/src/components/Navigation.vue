<template>
  <nav class="navbar is-transparent">
    <div class="navbar-brand">
      <router-link class="navbar-item" to="/">
        <img src="../assets/attlogo.svg" alt="Logo" width="50" height="28">
        <h1 class="title">swarmhub</h1>
      </router-link>
      <router-link class="navbar-item" to="/">Tests {{ activeTestsStr }}</router-link>
      <router-link class="navbar-item" to="/grids">Grids {{ activeGridsStr }}</router-link>
    </div>

    <div class="navbar-menu"> 
      <div class="navbar-start">
      
      </div>   
      <div class="navbar-end">
        <a class="navbar-item">
          {{ User }}
        </a>
          <a class="navbar-item" @click="logout();">
            Sign out
          </a>
      </div>
    </div>
  </nav>
</template>

<script>
import axios from 'axios';

export default {
  name: 'Navigation',
  data: function () {
    return {
      User: null
      }
  },
  props: {
    activeTests: Number,
    activeGrids: Number
  },
  computed: {
    activeTestsStr: function () {
      if (this.activeTests > 0) {
        return '(' + this.activeTests + ')'

      }
      return ''
    },
    activeGridsStr: function () {
      if (this.activeGrids > 0) {
        return '(' + this.activeGrids + ')'

      }
      return ''
    },
  },
  methods: {
    logout: function () {
      axios
      .post('logout')
      .then(response => (window.location = "/login"))
      .catch(error => {console.log("FAILURE: ", error)});
    }
  }
}
</script>
