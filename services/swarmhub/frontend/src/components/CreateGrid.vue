<template>
      <div class="modal" v-bind:class="{ 'is-active': isCreateGridModalActive}">
        <div class="modal-background"></div>
        <div class="modal-card">
          <header class="modal-card-head">
            <p class="modal-card-title">Create Grid</p>
            <button class="delete" aria-label="close" @click="$emit('is-active', false); clearGridSelection();"></button>
          </header>
          <section class="modal-card-body">
            <div class="field">
              <label class="label">Name</label>
              <div class="control">
                <input class="input" v-model="createGridData.Name" type="text" placeholder="Name of the Grid">
              </div>
            </div>
            <div class="field">
              <label class="label">Provider</label>
              <div class="control">
                <div class="select">
                  <select v-model="createGridData.Provider" @change="getProviderInfo(createGridData.Provider);">
                    <option :value="undefined" disabled style="display:none">Select Provider</option>
                    <option  v-for="provider in providers" :key="provider" :value="provider">{{provider}}</option>
                  </select>
                </div>
              </div>
            </div>
            <div class="field">
              <label class="label">Region</label>
              <div class="control">
                <div class="select">
                  <select v-bind:disabled="!createGridData.Provider" v-model="createGridData.Region" @change="getRegionInfo(createGridData.Provider, createGridData.Region);">
                    <option :value="undefined" disabled style="display:none">Select Region</option>
                    <option v-for="region in gridOptions.Region" :key="region" :value="region">{{region}}</option>
                  </select>
                </div>
              </div>
            </div>
            <div class="field">
              <label class="label">Master Instance Type</label>
              <div class="control">
                <div class="select">
                  <select v-bind:disabled="!createGridData.Region" v-model="createGridData.MasterType">
                    <option :value="undefined" disabled style="display:none">Select Master Type</option>
                    <option v-for="mastertype in gridOptions.MasterType" :key="mastertype" :value="mastertype">{{mastertype}}</option>
                  </select>
                </div>
              </div>
            </div>
            <div class="field">
              <label class="label">Slave Instance Type</label>
              <div class="control">
                <div class="select">
                  <select v-bind:disabled="!createGridData.Region" v-model="createGridData.SlaveType">
                    <option :value="undefined" disabled style="display:none">Select Slave Type</option>
                    <option v-for="slavetype in gridOptions.SlaveType" :key="slavetype" :value="slavetype">{{slavetype}}</option>
                  </select>
                </div>
              </div>
            </div>
            <div class="field">
              <label class="label">Number of Slave Nodes</label>
              <div class="control">
                <input v-bind:disabled="!createGridData.SlaveType" v-model="createGridData.SlaveNodes" class="input" type="number" placeholder="Number of slaves">
              </div>
            </div>
            <div class="field">
              <label class="label">TTL</label>
              <div class="control">
                <input v-model="createGridData.TTL" class="input" type="number" placeholder="Time in minutes until the grid is deleted.">
              </div>
            </div>
          </section>
          <footer class="modal-card-foot">
            <button class="button is-success" @click="createGrid(); $emit('is-active', false);">Save changes</button>
            <button class="button" @click="$emit('is-active', false); clearGridSelection();">Cancel</button>
          </footer>
        </div>
    </div>
</template>

<script>
import axios from 'axios';

export default {
  name: 'CreateGrid',
  props: {
    isCreateGridModalActive: Boolean,
    providers: Array,
  },
  data: function () {
    return {
      listOfTests: [],
      createGridData: {
        Name: '',
        Provider: undefined,
        Region: undefined,
        MasterType: undefined,
        SlaveType: undefined,
        SlaveNodes: '',
        TTL: ''
      },
      gridOptions: {
        Provider: [],
        Region: [],
        MasterType: [],
        SlaveType: []
      },
    }
  },
  methods: {
    createGrid: function() {
      var payload
      this.createGridData.SlaveNodes = parseInt(this.createGridData.SlaveNodes, 10)
      this.createGridData.TTL = parseInt(this.createGridData.TTL, 10)
      payload = JSON.parse(JSON.stringify(this.createGridData));
      axios
        .post('/api/grid', payload)
        .then(response => {
          this.$emit('get-grids', 'latest')
          if (response.status == 200) {
            this.clearGridSelection()
          }
        })
    },
    getProviderInfo: function(provider) {
      this.createGridData.Region = undefined
      this.createGridData.MasterType = undefined
      this.createGridData.SlaveType = undefined
      this.createGridData.SlaveNodes = ''
      this.getGridRegions(provider)
    },
    getRegionInfo: function(provider, region) {
      this.createGridData.MasterType = undefined
      this.createGridData.SlaveType = undefined
      this.createGridData.SlaveNodes = ''
      this.getGridInstances(provider, region)
    },
    clearGridSelection: function() {
      this.createGridData.Name = ''
      this.createGridData.Provider = undefined
      this.createGridData.Region = undefined
      this.createGridData.MasterType = undefined
      this.createGridData.SlaveType = undefined
      this.createGridData.SlaveNodes = ''
      this.createGridData.TTL = ''
    },
    extractGridRegions: function (regions) {
      var i
      for (i = 0; i < regions.length; i++) {
            this.gridOptions.Region.push(regions[i].Region)
          }
    },
    getGridRegions: function (provider) {
      this.gridOptions.Region = []
      axios
       .get('/api/grids/regions?provider=' + provider)
       .then(response => this.extractGridRegions(response.data.Regions))
    },
    extractGridInstances: function (instances) {
      var i
      for (i = 0; i < instances.length; i++) {
        this.gridOptions.MasterType.push(instances[i].Instance)
        this.gridOptions.SlaveType.push(instances[i].Instance)
      }
    },
    getGridInstances: function (provider, region) {
      this.gridOptions.MasterType = []
      this.gridOptions.SlaveType = []
      axios
       .get('/api/grids/instances?provider=' + provider + '&region=' + region)
       .then(response => this.extractGridInstances(response.data.Instances))
    },
  }
}
</script>