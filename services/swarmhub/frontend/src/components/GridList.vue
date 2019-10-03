<template>
  <div class="tile is-7 is-vertical is-parent">
    <div class="tile is-child box">
      <div class="level">
        <div class="level-left">
          <a class="button is-info title is-5" @click="isCreateGridModalActive = true; getGridProviders();">Create Grid</a>
        </div>
      </div>
      <div class="box">
        <div class="table-container">
          <table class="table is-hoverable" style="width:100%" >
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Health</th>
              <th>Provider</th>
              <th>Region</th>
              <th>Master Type</th>
              <th>Slave Type</th>
              <th>Slave Nodes</th>
              <th>TTL</th>
            </tr>
            <tr v-for="grid in listOfGrids" :key="grid.ID" v-bind:class="{ 'is-selected': selectedGridID==grid.ID }" @click="$emit('selected-grid', grid.ID);"> 
              <td>{{ grid.Name }}</td>
              <td>{{ grid.Status }}</td>
              <td>{{ grid.Health }}</td>
              <td>{{ grid.Provider }}</td>
              <td>{{ grid.Region }}</td>
              <td>{{ grid.Master }}</td>
              <td>{{ grid.Slave }}</td>
              <td>{{ grid.Nodes }}</td>
              <td>{{ grid.TTL }}</td>
            </tr>
          </table> 
        </div>
      </div>
      <nav class="pagination is-right" role="navigation" aria-label="pagination">
          <button :disabled="firstGridID===firstGridIDInList" class="pagination-previous" @click="$emit('previous-page', firstGridIDInList);">Previous</button>
          <button :disabled="lastGridID===lastGridIDInList" class="pagination-next" @click="$emit('next-page', lastGridIDInList);">Next page</button>
      </nav>
      <create-grid :isCreateGridModalActive="isCreateGridModalActive" :gridID="selectedGridID" :providers="providers" @is-active="updateCreateGridBool" @get-grids="getGrids"/>
      </div> 
    </div>
</template>

<script>
import axios from 'axios';
import CreateGrid from '@/components/CreateGrid.vue'

export default {
  name: 'GridList',
  components: {
    CreateGrid,
  },
  props: {
    selectedGridID: String,
    listOfGrids: Array,
    firstGridIDInList: String,
    lastGridIDInList: String,
    firstGridID: String,
    lastGridID: String
  },
  data: function () {
    return {
      isCreateGridModalActive: false,
      providers: [],
    }
  },
  methods: {
    updateCreateGridBool (bool) {
      this.isCreateGridModalActive = bool
    },
    getGridProviders: function () {
      axios
       .get('/api/grids/providers')
       .then(response => this.providers = response.data.Providers)
       .catch(error => {console.log("FAILURE: ", error)});
    },
    getGrids: function (gridID) {
      this.$emit('get-grids', gridID)
    },
  },
}
</script>