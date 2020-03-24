<template>
  <div class="tile is-parent">
    <div v-if="grid" class="tile is-child box">
      <h1 class="title">{{ grid.Name }}</h1>
      <p class="subtitle is-6">
        <button
          class="button is-small is-rounded is-danger"
          :disabled="grid.Status === 'Deploying'"
          @click="isDeleteGridModalActive = true;"
        >delete</button>
      </p>
      <div class="block">
        <p class="header is-6">
          Status: {{ grid.Status }} (
          <a @click="showLogs(true);">show logs</a>)
        </p>
        <p class="header is-6">Health: {{ grid.Health }}</p>
        <p class="header is-6">Created: {{ grid.Created }}</p>
      </div>
      <div class="gridDetails">
        <div class="block">
          <p>{{ grid.Desc }}</p>
        </div>
      </div>
      <div class="block">
        <button :disabled="isLaunchDisabled" class="button is-link" @click="deployGrid();">Launch</button>
      </div>

      <div class="modal" v-bind:class="{ 'is-active': isDeleteGridModalActive }">
        <div class="modal-background"></div>
        <div class="modal-content">
          <div class="box">
            <p>Are you sure you want to delete {{ grid.Name }}?</p>
            <button
              class="button is-danger"
              @click="isDeleteGridModalActive = false; deleteGrid(grid.ID);"
            >Delete</button>
            <button class="button" @click="isDeleteGridModalActive = false;">Cancel</button>
          </div>
        </div>
      </div>

      <div class="modal" v-bind:class="{ 'is-active': isShowLogsModalActive}">
        <div class="modal-background"></div>
        <div class="modal-card">
          <header class="modal-card-head">
            <p class="modal-card-title">Logs for {{grid.Name}}</p>
            <button class="delete" aria-label="close" @click="showLogs(false);"></button>
          </header>
          <section class="modal-card-body">
            <div class="content">
              <h5>Currently Deploying: {{logStatus}}</h5>
            </div>
            <p v-for="(log, index) in logs" :key="index">{{ logPrint(log) }}</p>
          </section>
          <footer class="modal-card-foot">
            <button class="button" @click="showLogs(false);">Close</button>
          </footer>
        </div>
      </div>
    </div>
    <div v-else class="tile is-child box">
      <h1 class="title">No grids selected</h1>
    </div>
  </div>
</template>

<script>
import axios from "axios";
import { printLogMixin } from '../mixins/printLogMixin'

export default {
  name: "GridDetails",
  mixins: [printLogMixin],
  props: {
    gridID: String,
    currentGridStatus: String
  },
  data: function() {
    return {
      grid: null,
      isDeleteGridModalActive: false,
      isShowLogsModalActive: false,
      gettingLogs: false,
      logStatus: "",
      logs: "",
      launchCLicked: false
    };
  },
  beforeMount() {
    this.loadGridData(this.gridID);
    if (this.$route.name == "gridlogs") {
      this.showLogs(true);
    }
  },
  computed: {
    isLaunchDisabled: function() {
      if (this.launchClicked == true) {
        return true;
      }

      if (this.grid != null && this.grid.Status == "Ready") {
        return false;
      }

      return true;
    }
  },
  watch: {
    gridID: function(val) {
      this.loadGridData(val);
      this.launchClicked = false;
    },
    currentGridStatus: function() {
      // adding if statement to prevent duplicate API calls on changes.
      if (this.currentGrid == this.gridID) {
        this.loadGridData(this.gridID);
      } else {
        this.currentGrid = this.gridID;
      }
    }
  },
  methods: {
    deleteGrid: function(id) {
      console.log("Delete id: ", id);
      axios
        .post("/api/grid/" + id + "/delete")
        .then(() => this.$emit("get-grids"))
        .then(this.$router.push("/grids"));
    },
    loadGridData: function(gridid) {
      if (gridid) {
        axios
          .get("/api/grid/" + gridid)
          .then(response => (this.grid = response.data))
          .catch(error => {
            console.log("FAILURE: ", error);
          });
      } else {
        this.grid = null;
      }
    },
    getLog: function(id) {
      if (
        this.logs.length > 0 &&
        this.logs[this.logs.length - 1].Running == true
      ) {
        axios
          .get("/api/test/" + id + "/deploylogs")
          .then(response => (this.logs = response.data))
          .then((this.logStatus = this.logs[this.logs.length - 1].Running));
      } else {
        clearInterval(this.gettingLogs);
        this.logStatus = false;
      }
    },
    getLogs: function(id) {
      axios.get("/api/test/" + id + "/deploylogs").then(response => {
        this.logs = response.data;
        this.logStatus = response.data[response.data.length - 1].Running;
        this.gettingLogs = setInterval(() => {
          this.getLog(id);
        }, 3000);
      });
    },
    stopGettingLogs: function() {
      this.$router.push("/grids/" + this.gridID);
      this.logs = "";
      clearInterval(this.gettingLogs);
      this.logStatus = "";
    },
    showLogs: function(bool) {
      if (bool == true) {
        this.getLogs(this.gridID);
        this.$router.push("/grids/" + this.gridID + "/logs");
        this.isShowLogsModalActive = true;
      } else {
        this.$router.push("/grids/" + this.gridID);
        this.stopGettingLogs();
        this.isShowLogsModalActive = false;
      }
    },
    deployGrid: function() {
      var path;
      path = "/api/grid/" + this.grid.ID + "/start";
      axios.post(path).then(() => {
        this.loadGridData(this.grid.ID);
        this.$emit("get-grids");
      });
    }
  }
};
</script>
