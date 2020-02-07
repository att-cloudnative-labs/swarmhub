<template>
  <div class="modal" v-bind:class="{ 'is-active': isGridTemplateModalActive}">
    <div class="modal-background"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Grid Templates</p>
        <button
          class="delete"
          aria-label="close"
          @click="$emit('is-active', false); clearGridSelection();"
        ></button>
      </header>
      <section class="modal-card-body" style="min-height: 0px">
        <div class="field is-horizontal">
          <div class="field-label is-normal">
            <label class="label">Choose:</label>
          </div>
          <div class="field-body">
            <div class="field is-narrow">
              <div class="control">
                <div class="buttons">
                  <button
                    v-bind:class="{'button is-link': createNewClicked, 'button': !createNewClicked}"
                    v-on:click="handleSelection('createNew'); clearGridSelection()"
                  >Create New Template</button>
                  <button
                    v-bind:class="{'button is-link': selectExistingClicked, 'button': !selectExistingClicked}"
                    v-on:click="handleSelection('selectExisting'); clearGridSelection()"
                  >Existing Templates</button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <hr />

        <div v-if="createNewClicked">
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">Name</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <input class="input" type="text" v-model=" createGridData.Name" />
                </div>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">Provider</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <div class="select">
                    <select
                      v-model="createGridData.Provider"
                      @change="getProviderInfo(createGridData.Provider);"
                    >
                      <option :value="undefined" disabled style="display:none">Select Provider</option>
                      <option
                        v-for="provider in providers"
                        :key="provider"
                        :value="provider"
                      >{{provider}}</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">Region</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <div class="select">
                    <select
                      v-bind:disabled="!createGridData.Provider"
                      v-model="createGridData.Region"
                      @change="getRegionInfo(createGridData.Provider, createGridData.Region);"
                    >
                      <option :value="undefined" disabled style="display:none">Select Region</option>
                      <option
                        v-for="region in gridOptions.Region"
                        :key="region"
                        :value="region"
                      >{{region}}</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">Master Type</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <div class="select">
                    <select
                      v-bind:disabled="!createGridData.Region"
                      v-model="createGridData.MasterType"
                    >
                      <option :value="undefined" disabled style="display:none">Select Master Type</option>
                      <option
                        v-for="mastertype in gridOptions.MasterType"
                        :key="mastertype"
                        :value="mastertype"
                      >{{mastertype}}</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">Slave Type</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <div class="select">
                    <select
                      v-bind:disabled="!createGridData.Region"
                      v-model="createGridData.SlaveType"
                    >
                      <option :value="undefined" disabled style="display:none">Select Slave Type</option>
                      <option
                        v-for="slavetype in gridOptions.SlaveType"
                        :key="slavetype"
                        :value="slavetype"
                      >{{slavetype}}</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">Slave Nodes</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <input
                    v-bind:disabled="!createGridData.Region"
                    v-model="createGridData.SlaveNodes"
                    class="input"
                    type="number"
                    placeholder="Number of slaves"
                  />
                </div>
              </div>
            </div>
          </div>
          <div class="field is-horizontal">
            <div class="field-label is-normal">
              <label class="label">TTL</label>
            </div>
            <div class="field-body">
              <div class="field is-narrow">
                <div class="control">
                  <input
                    v-model="createGridData.TTL"
                    class="input"
                    type="number"
                    placeholder="Time in minutes until the grid is deleted."
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="selectExistingClicked">
          <div class="field is-horizontal">
            <label class="label" style="padding:5px 10px 0 0">Select template:</label>
            <div class="control">
              <div class="select">
                <select v-model="selectedTemplate" @change="handleSelectTemplate()">
                  <option :value="undefined" disabled style="display:none">Select template</option>
                  <option
                    v-for="template in gridTemplates"
                    :key="template.ID"
                    :value="template"
                  >{{template.Name}}</option>
                </select>
              </div>
            </div>
            <div v-if="templateSelected">
              <div class="control" style="padding-left:15px">
                <button class="button is-white" @click="editTemplateActive=true">
                  <span class="icon">
                    <font-awesome-icon icon="edit" />
                  </span>
                </button>
                <button
                  class="button is-white"
                  @click="editTemplateActive=false; updateGridTemplate()"
                >
                  <span class="icon has-text-info">
                    <font-awesome-icon icon="save" />
                  </span>
                </button>
                <button class="button is-white" @click="deleteGridTemplate()">
                  <span class="icon has-text-danger">
                    <font-awesome-icon icon="trash" />
                  </span>
                </button>
              </div>
            </div>
          </div>

          <hr />

          <div v-if="templateSelected">
            <fieldset v-bind:disabled="!editTemplateActive">
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">Name</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <input class="input" type="text" v-model="selectedTemplate.Name" />
                    </div>
                  </div>
                </div>
              </div>
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">Provider</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <div class="select">
                        <select
                          v-model="createGridData.Provider"
                          @change="getProviderInfo(createGridData.Provider);"
                        >
                          <option
                            :value="createGridData.Provider"
                            disabled
                          >{{createGridData.Provider}}</option>
                          <option
                            v-for="provider in providers"
                            :key="provider"
                            :value="provider"
                          >{{provider}}</option>
                        </select>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">Region</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <div class="select">
                        <select
                          v-model="createGridData.Region"
                          @change="getRegionInfo(createGridData.Provider, createGridData.Region);"
                        >
                          <option :value="createGridData.Region" disabled>{{createGridData.Region}}</option>
                          <option
                            v-for="region in gridOptions.Region"
                            :key="region"
                            :value="region"
                          >{{region}}</option>
                        </select>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">Master Type</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <div class="select">
                        <select v-model="createGridData.MasterType">
                          <option
                            :value="createGridData.MasterType"
                            disabled
                          >{{createGridData.MasterType}}</option>
                          <option
                            v-for="mastertype in gridOptions.MasterType"
                            :key="mastertype"
                            :value="mastertype"
                          >{{mastertype}}</option>
                        </select>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">Slave Type</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <div class="select">
                        <select v-model="createGridData.SlaveType">
                          <option
                            :value="createGridData.SlaveType"
                            disabled
                          >{{createGridData.SlaveType}}</option>
                          <option
                            v-for="slavetype in gridOptions.SlaveType"
                            :key="slavetype"
                            :value="slavetype"
                          >{{slavetype}}</option>
                        </select>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">Slave Nodes</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <input class="input" type="text" v-model="selectedTemplate.SlaveNodes" />
                    </div>
                  </div>
                </div>
              </div>
              <div class="field is-horizontal">
                <div class="field-label is-normal">
                  <label class="label">TTL</label>
                </div>
                <div class="field-body">
                  <div class="field is-narrow">
                    <div class="control">
                      <input class="input" type="text" v-model="selectedTemplate.TTL" />
                    </div>
                  </div>
                </div>
              </div>
            </fieldset>
          </div>
        </div>
      </section>
      <footer class="modal-card-foot">
        <button
          v-if="createNewClicked"
          class="button is-success"
          @click="createGridTemplate(); $emit('is-active', false);"
        >Create</button>
        <button
          v-if="selectExistingClicked"
          class="button is-success"
          @click="createGrid(); $emit('is-active', false);"
        >Proceed</button>
        <button class="button" @click="$emit('is-active', false); clearGridSelection()">Cancel</button>
      </footer>
    </div>
  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "GridTemplates",
  props: {
    isGridTemplateModalActive: Boolean,
    providers: Array,
    gridTemplates: Array
  },
  data: function() {
    return {
      createGridData: {
        Name: "",
        Provider: undefined,
        Region: undefined,
        MasterType: undefined,
        SlaveType: undefined,
        SlaveNodes: "",
        TTL: ""
      },
      gridOptions: {
        Provider: [],
        Region: [],
        MasterType: [],
        SlaveType: []
      },
      createNewClicked: false,
      selectExistingClicked: false,
      templateSelected: false,
      selectedTemplate: {},
      editTemplateActive: false
    };
  },
  methods: {
    createGrid: function() {
      var payload;
      this.createGridData.SlaveNodes = parseInt(
        this.createGridData.SlaveNodes,
        10
      );
      this.createGridData.TTL = parseInt(this.createGridData.TTL, 10);
      payload = JSON.parse(JSON.stringify(this.createGridData));
      axios.post("/api/grid", payload).then(response => {
        this.$emit("get-grids", "latest");
        if (response.status == 200) {
          this.clearGridSelection();
        }
      });
      this.createNewClicked = false;
    },
    createGridTemplate: function() {
      var payload;
      this.createGridData.SlaveNodes = parseInt(
        this.createGridData.SlaveNodes,
        10
      );
      this.createGridData.TTL = parseInt(this.createGridData.TTL, 10);
      payload = JSON.parse(JSON.stringify(this.createGridData));
      axios.post("/api/grid_template", payload).then(response => {
        this.$emit("get-grids", "latest");
        this.$emit("get-gridTemplates", "latest");
        if (response.status == 201) {
          this.clearGridSelection();
        }
      });
      this.selectExistingClicked = false;
    },
    updateGridTemplate: function() {
      var payload;
      this.createGridData.SlaveNodes = parseInt(
        this.createGridData.SlaveNodes,
        10
      );
      this.createGridData.TTL = parseInt(this.createGridData.TTL, 10);
      payload = JSON.parse(JSON.stringify(this.createGridData));
      axios
        .put("/api/grid_template/" + this.selectedTemplate.ID, payload)
        .then(response => {
          this.$emit("get-grids", "latest");
          this.$emit("get-gridTemplates", "latest");
        });
    },
    deleteGridTemplate: function() {
      axios
        .delete("/api/grid_template/" + this.selectedTemplate.ID)
        .then(response => {
          this.$emit("get-grids", "latest");
          this.$emit("get-gridTemplates", "latest");
          if (response.status == 204) {
            this.clearGridSelection();
          }
        });
      this.clearGridSelection();
      this.selectExistingClicked = false;
    },
    getProviderInfo: function(provider) {
      this.createGridData.Region = undefined;
      this.createGridData.MasterType = undefined;
      this.createGridData.SlaveType = undefined;
      this.createGridData.SlaveNodes = "";
      this.getGridRegions(provider);
    },
    getRegionInfo: function(provider, region) {
      this.createGridData.MasterType = undefined;
      this.createGridData.SlaveType = undefined;
      this.createGridData.SlaveNodes = "";
      this.getGridInstances(provider, region);
    },
    clearGridSelection: function() {
      this.createGridData.Name = "";
      this.createGridData.Provider = undefined;
      this.createGridData.Region = undefined;
      this.createGridData.MasterType = undefined;
      this.createGridData.SlaveType = undefined;
      this.createGridData.SlaveNodes = "";
      this.createGridData.TTL = "";
      this.templateSelected = false;
      this.selectedTemplate = {};
      this.editTemplateActive = false;
    },
    extractGridRegions: function(regions) {
      var i;
      for (i = 0; i < regions.length; i++) {
        this.gridOptions.Region.push(regions[i].Region);
      }
    },
    getGridRegions: function(provider) {
      this.gridOptions.Region = [];
      axios
        .get("/api/grids/regions?provider=" + provider)
        .then(response => this.extractGridRegions(response.data.Regions));
    },
    extractGridInstances: function(instances) {
      var i;
      for (i = 0; i < instances.length; i++) {
        this.gridOptions.MasterType.push(instances[i].Instance);
        this.gridOptions.SlaveType.push(instances[i].Instance);
      }
    },
    getGridInstances: function(provider, region) {
      this.gridOptions.MasterType = [];
      this.gridOptions.SlaveType = [];
      axios
        .get("/api/grids/instances?provider=" + provider + "&region=" + region)
        .then(response => this.extractGridInstances(response.data.Instances));
    },
    handleSelection: function(selection) {
      this.createGridData = {};
      if (selection === "createNew") {
        this.createNewClicked = !this.createNewClicked;
        this.selectExistingClicked = !this.createNewClicked;
      } else if (selection === "selectExisting") {
        this.selectExistingClicked = !this.selectExistingClicked;
        this.createNewClicked = !this.selectExistingClicked;
      }
    },
    handleSelectTemplate: function() {
      this.templateSelected = true;
      this.createGridData = this.selectedTemplate;
      this.getGridRegions(this.createGridData.Provider);
      this.getGridInstances(this.createGridData.Provider, this.createGridData.Region);
    }
  }
};
</script>

