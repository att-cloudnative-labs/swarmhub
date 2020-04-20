<template>
  <div id="app">
    <div id="nav">
      <Navigation :activeTests="activeTests" :activeGrids="activeGrids" />
    </div>
    <router-view @get-tests="getActiveTests" @get-grids="getActiveGrids" />
  </div>
</template>

<script>
import "@/assets/styles.css";
import Navigation from "./components/Navigation";
export default {
  name: "app",
  components: {
    Navigation: Navigation
  },
  data: function() {
    return {
      activeTests: 0,
      activeGrids: 0
    };
  },
  beforeMount() {
    this.getActiveTests();
    this.getActiveGrids();
  },
  methods: {
    getActiveTests: function() {
      fetch("/api/status/test?status=Deployed")
        .then(response => response.json())
        .then(data => (this.activeTests = data.length));
    },
    getActiveGrids: function() {
      fetch("/api/status/grid?status=Deployed&status=Available")
        .then(response => response.json())
        .then(data => {
          console.log(
            "getActiveGrids was activated, length is: " + data.length
          );
          this.activeGrids = data.length;
        });
    }
  }
};
</script>

<style>
#app {
  font-family: "Avenir", Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}
#nav {
  padding: 30px;
  padding-bottom: 0px;
}

#nav a {
  font-weight: bold;
  color: #2c3e50;
}

#nav a.router-link-exact-active {
  color: #42b983;
}
</style>
