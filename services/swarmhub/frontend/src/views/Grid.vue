<template>
  <div class="grid">
    <section class="section">
      <div class="tile is-ancestor">
        <GridList
          :selectedGridID="id"
          :firstGridID="paginateInfo.FirstGrid"
          :lastGridID="paginateInfo.LastGrid"
          :firstGridIDInList="firstGridIDInList"
          :lastGridIDInList="lastGridIDInList"
          :listOfGrids="listOfGrids"
          @selected-grid="updateSelectedGrid"
          @get-grids="getGrids"
          @previous-page="getPreviousPage"
          @next-page="getNextPage"
        />
        <GridDetails :gridID="id" :currentGridStatus="currentGridStatus" @get-grids="getGrids" />
      </div>
    </section>
  </div>
</template>

<script>
// @ is an alias to /src
import axios from "axios";
import GridList from "@/components/GridList.vue";
import GridDetails from "@/components/GridDetails.vue";

export default {
  name: "grid",
  components: {
    GridList,
    GridDetails
  },
  props: {
    id: {
      type: String,
      default: ""
    }
  },
  data: function() {
    return {
      listOfGrids: [],
      gettingGridsLoopActive: false,
      currentGridStatus: null,
      firstGridIDInList: "",
      lastGridIDInList: "",
      itemsInList: 10,
      paginateInfo: { FirstGrid: "", LastGrid: "", NumberOfGrids: 0 }
    };
  },
  beforeMount() {
    this.getGrids();
  },
  methods: {
    updateSelectedGrid: function(id) {
      this.$router.push("/grids/" + id);
    },
    updateCurrentGridStatus: function() {
      if (this.listOfGrids == null) {
        return;
      }
      var i;
      for (i = 0; i < this.listOfGrids.length; i++) {
        if (
          this.id == this.listOfGrids[i].ID &&
          this.currentGridStatus != this.listOfGrids[i].Status
        ) {
          this.currentGridStatus = this.listOfGrids[i].Status;
        }
      }
    },
    loadGridInfo: function() {
      axios
        .get("/api/paginate/grid/info")
        .then(response => (this.paginateInfo = response.data))
        .catch(error => {
          console.log("FAILURE: ", error);
        });
    },
    getPreviousPage: function(gridID) {
      console.log("getPreviousPage");
      axios
        .get(
          "/api/paginate/grid/key/" + gridID + "?offset=-" + this.itemsInList
        )
        .then(response => {
          // i would think i wouldn't need to update id because it would get updated by
          // the prop, doesn't appear to be the case.
          this.id = response.data;
          this.$router.push("/grids/" + response.data, this.getGrids());
        })
        .catch(error => {
          console.log("FAILURE: ", error);
        });
    },
    getNextPage: function(testID) {
      console.log("getNextPage");
      axios
        .get("/api/paginate/grid/key/" + testID + "?offset=1")
        .then(response => {
          // i would think i wouldn't need to update id because it would get updated by
          // the prop, doesn't appear to be the case.
          this.id = response.data;
          this.$router.push("/grids/" + response.data, this.getGrids());
        })
        .catch(error => {
          console.log("FAILURE: ", error);
        });
    },
    getPaginateInfo: function() {
      axios
        .get("/api/paginate/grid/info")
        .then(response => (this.paginateInfo = response.data));
    },
    getGridsAPICall: function(url) {
      axios.get(url).then(response => {
        if (response.data == null || response.data.length == 0) {
          this.listOfGrids = [];
          this.firstGridIDInList = "";
          this.lastGridIDInList = "";
        } else {
          this.listOfGrids = response.data;
          this.firstGridIDInList = response.data[0].ID;
          this.lastGridIDInList = response.data[response.data.length - 1].ID;
        }

        if (this.gettingGridsLoopActive == false) {
          this.gettingGridsLoopActive = true;
          setTimeout(this.getGridsLoop, 5000);
        }
        this.updateCurrentGridStatus();
      });
    },
    getGrids: function(gridID) {
      console.log("current id is ", this.id);
      this.getPaginateInfo();
      var url = "/api/grids";
      var urlExtra;
      if (gridID == "latest") {
        urlExtra = "";
      } else if (gridID != null && gridID != "") {
        console.log("using testID passed in function ", gridID);
        urlExtra = "/list/" + gridID;
      } else if (this.id == "") {
        console.log("getting tests with no id selected.");
        urlExtra = "";
      } else if (
        this.listOfGrids.filter(grid => grid.ID === this.id).length === 1
      ) {
        console.log("getting tests when the test is in the current list");
        urlExtra = "/list/" + this.firstGridIDInList;
      } else {
        console.log("Getting the test with new set of data.");
        urlExtra = "/list/" + this.id;
      }

      url = url + urlExtra;
      this.getGridsAPICall(url);
      this.$emit("get-grids", "");
    },
    getGridsLoop: function() {
      if (!Array.isArray(this.listOfGrids)) {
        this.gettingGridsLoopActive = false;
        return;
      }

      var tryAgain = false;
      var i;
      for (i = 0; i < this.listOfGrids.length; i++) {
        var status = this.listOfGrids[i].Status;
        // adding logic to check if the status of the currently selected test has
        // updated. If it has then the test details needs to make an api call to get the latest.
        if (
          this.id == this.listOfGrids[i].ID &&
          this.currentGridStatus != status
        ) {
          this.currentGridStatus = status;
        }
        if (status == "Deploying") {
          tryAgain = true;
        }
      }
      if (tryAgain == true) {
        this.getGrids();
        setTimeout(this.getGridsLoop, 5000);
        return;
      }
      this.gettingGridsLoopActive = false;
    }
  }
};
</script>
