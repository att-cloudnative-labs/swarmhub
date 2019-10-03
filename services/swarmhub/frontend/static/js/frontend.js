var app = new Vue({
  delimiters: ['{(', ')}'],
  el: '#app',
  data: {
          uploadPercentage: 0,
          highlightRow: true,
          data: false,
          gettingLogs: false,
          logStatus: '',
          logs: '',
          listOfTests: [],
          listOfGrids: [],
          isCreateGridModalActive: false,
          isCreateTestModalActive: false,
          isShowLogsModalActive: false,
          isLaunchTestModalActive: false,
          gridIDForTest: undefined,
          isAutomatic: false,
          isDeleteGridModalActive: false,
          createTestData: {
              Name: '',
              Desc: '',
          },
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
  },
  computed: {
    deployedGrids: function() {
      return this.listOfGrids.filter(function(g) {
        return g.Status == "Deployed"
      })
    }
  },
  mounted () {
    currentUrl = window.location.pathname
    if ( currentUrl == "/tests" || currentUrl == "/") {
      this.getAllTests()
    } else if ( currentUrl == "/grids") {
      this.getGrids()
    }
  },
  methods: {
    getAllTests: function () {
      axios
      .get('/api/tests')
      .then(response => (this.listOfTests = response.data))
    },
    launchTest: function (testID, gridID, isAutomatic) {
      console.log("Launching test: ", testID, gridID, isAutomatic)
    },
    deleteGrid: function(id) {
      console.log("Delete id: ", id)
      axios
      .post('/api/grid/' + id + '/delete')
      .then(response => this.getGrids())
    },
    getGrids: function () {
      axios
      .get('/api/grids')
      .then(response => (this.listOfGrids = response.data))
    },
    getGridProviders: function () {
      axios
       .get('/api/grids/providers')
       .then(response => this.gridOptions.Provider = response.data.Providers)
    },
    extractGridRegions: function (regions) {
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
    loadTestData: function (testid) {
      axios
        .get('/api/test?id=' + testid)
        .then(response => (this.data = response.data))     
    },
    loadGridData: function (gridid) {
      axios
        .get('/api/grid/' + gridid)
        .then(response => (this.data = response.data))     
    },
    clearCreateTest: function(){
      this.createTestData.Name = '';
      this.createTestData.Desc = '';
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
    handleFileUpload: function() {
      this.file = document.getElementById('fileid').files[0];
    },
    createTest: function() {
      formData = new FormData();
      formData.append("file", this.file);
      formData.append("metadata", JSON.stringify(this.createTestData));
      axios
        .post('/api/test', formData, {
          headers: {'Content-Type': 'multipart/form-data'},
          onUploadProgress: function( progressEvent ) {
            this.uploadPercentage = parseInt( Math.round( ( progressEvent.loaded * 100 ) / progressEvent.total ) );
          }.bind(this)
        })
        .then(this.getAllTests())
        .catch(error => {console.log("FAILURE: ", error)});
      this.clearCreateTest()
    },
    getLog: function(testid){
      if(this.logs.length > 0 && this.logs[this.logs.length-1].Running == true) {
      axios
        .get("/api/test/" + testid + "/deploylogs")
        .then(response => (this.logs = response.data))
        .then(this.logStatus = this.logs[this.logs.length-1].Running) 
      }else{
        clearInterval(this.gettingLogs)
        this.logStatus = false
      }
    },
    getLogs: function(testid){
      axios
        .get("/api/test/" + testid + "/deploylogs")
        .then(response => {
          this.logs = response.data
          this.logStatus = response.data[response.data.length-1].Running
          this.gettingLogs = setInterval(() => {this.getLog(testid)}, 3000)
        })
    },
    stopGettingLogs: function() {
      this.logs = ''
      clearInterval(this.gettingLogs)
      this.logStatus = ''
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
    createGrid: function() {
      this.createGridData.SlaveNodes = parseInt(this.createGridData.SlaveNodes, 10)
      this.createGridData.TTL = parseInt(this.createGridData.TTL, 10)
      payload = JSON.parse(JSON.stringify(this.createGridData));
      axios
        .post('/api/grid', payload)
        .then(this.getGrids())
    }
  }
})
