<template>
  <div class="home">
    <section class="section">
      <div class="tile is-ancestor">
        <TestList :selectedTestID="id" :firstTestID="paginateInfo.FirstTest" :lastTestID="paginateInfo.LastTest" :firstTestIDInList="firstTestIDInList" :lastTestIDInList="lastTestIDInList"  :listOfTests="listOfTests" @selected-test="updateSelectedTest" @get-tests="getTests" @previous-page="getPreviousPage" @next-page="getNextPage" @get-tests-fields="getTestsFields"/>
        <TestDetails :testID="id" :currentTestStatus="currentTestStatus" @get-tests="getTests"/>
      </div>
    </section>
  </div>
</template>

<script>
// @ is an alias to /src
import axios from 'axios';
import TestList from '@/components/TestList.vue'
import TestDetails from '@/components/TestDetails.vue'

export default {
  name: 'home',
  components: {
    TestList,
    TestDetails
  },
  data: function () {
    return {
      listOfTests: [],
      firstTestIDInList: "",
      lastTestIDInList: "",
      itemsInList: 10,
      gettingTestsLoopActive: false,
      currentTestStatus: null,
      paginateInfo: {FirstTest: "", LastTest: "", NumberOfTests: 0},
      startDate: "",
      endDate: "",
      searchField: "",
    }
  },
  props: {
    id: {
      type: String,
      default: "",
    },
  },
  watch: {
    listOfTests: function () {
      this.loadTestInfo()
    },
  },
  beforeMount() {
    this.getTests()
  },
  methods: {
    updateSelectedTest: function(id) {
      this.$router.push('/tests/' + id)
      this.updateCurrentTestStatus()
    },
    updateCurrentTestStatus: function() {
      if (this.listOfTests == null) {
        return
      }
      var i
      for (i = 0; i < this.listOfTests.length; i++) {
        if (this.id == this.listOfTests[i].ID && this.currentTestStatus != this.listOfTests[i].Status) {
          this.currentTestStatus = this.listOfTests[i].Status
        }
      }
    },
    loadTestInfo: function () {
      var url = '/api/paginate/test/info'
      var seperator = "?"
      if (this.startDate != "" && this.endDate != "") {
        seperator = "&"
        url = url + "?startdate=" + this.startDate + "&enddate=" + this.endDate
      }
      if (this.searchField != "") {
        url = url + seperator + "search=" + this.searchField
      }
      axios
        .get(url)
        .then(response => (this.paginateInfo = response.data))
        .catch(error => {console.log("FAILURE: ", error)});
    },
    getPreviousPage: function(testID) {
      console.log("getPreviousPage")
      var search = ""
      if (this.searchField != "") {
        search = "&search=" + this.searchField
      }
      axios
        .get('/api/paginate/test/key/' + testID + '?offset=-' + this.itemsInList + search)
        .then(response => {
          // i would think i wouldn't need to update id because it would get updated by
          // the prop, doesn't appear to be the case.
          this.id = response.data
          this.$router.push('/tests/' + response.data, this.getTests())
        })
        .catch(error => {console.log("FAILURE: ", error)}); 
    },
    getNextPage: function(testID) {
      console.log("getNextPage")
      var search = ""
      if (this.searchField != "") {
        search = "&search=" + this.searchField
      }
      axios
        .get('/api/paginate/test/key/' + testID + '?offset=1' + search)
        .then(response => {
          // i would think i wouldn't need to update id because it would get updated by
          // the prop, doesn't appear to be the case.
          this.id = response.data
          this.$router.push('/tests/' + response.data, this.getTests())
        })
        .catch(error => {console.log("FAILURE: ", error)}); 
    },
    getPaginateInfo: function () {
      var url = "/api/paginate/test/info"
      if (this.startDate != "" && this.endDate != "") {
        url = url + "?startdate=" + this.startDate + "&enddate=" + this.endDate
      }
      axios
        .get(url)
        .then(response => (this.paginateInfo = response.data))
    },
    getTestsFields: function (search, startDate, endDate) {
      console.log("start date is: " + startDate)
      console.log("end date is: " + endDate)
      this.searchField = search
      this.startDate = startDate
      this.endDate = endDate
      this.getTests()
    },
    getTestsAPICall: function (url) {
      axios
        .get(url)
        .then(response => {
          if (response.data == null || response.data.length == 0) {
            this.listOfTests = []
            this.firstTestIDInList = ""
            this.lastTestIDInList = ""
          } else {
            this.listOfTests = response.data
            this.firstTestIDInList = response.data[0].ID
            this.lastTestIDInList = response.data[response.data.length - 1].ID
          }
          
        if (this.gettingTestsLoopActive == false) {
          this.gettingTestsLoopActive = true
          setTimeout(this.getTestsLoop, 5000)
        }
        this.updateCurrentTestStatus()
      })
    },
    urlAddFields: function () {
      var u = ""
      var seperator = "?"
      if (this.startDate != "" && this.endDate != "") {
        seperator = "&"
        u = "?startdate=" + this.startDate + "&enddate=" + this.endDate
      }
      if (this.searchField != "") {
        u = u + seperator + "search=" + this.searchField
      }
      
      return u
    },
    getTests: function (testID) {
      console.log("current id is ", this.id)
      this.getPaginateInfo()
      var url = '/api/tests'
      var urlExtra
      if (testID == "latest") {
        urlExtra = ''
      } else if (testID != null && testID != "") {
        console.log("using testID passed in function ", this.testID)
        urlExtra = '/' + testID
      } else if (this.id == "") {
        console.log("getting tests with no id selected.")
        urlExtra = ''
      } else if (this.listOfTests.filter(test => test.ID === this.id ).length === 1) {
        console.log("getting tests when the test is in the current list")
        urlExtra = '/' + this.firstTestIDInList
      } else {
        console.log("Getting the test with new set of data.")
        urlExtra = '/' + this.id
      }

      url = url + urlExtra + this.urlAddFields()
      this.getTestsAPICall(url)
      this.$emit('get-tests', "")
    },
    getTestsLoop: function () {
      if (!Array.isArray(this.listOfTests)) {
        this.gettingTestsLoopActive = false
        return
      }

      var tryAgain = false
      var i
      for (i = 0; i < this.listOfTests.length; i++) {
        var status = this.listOfTests[i].Status
        // adding logic to check if the status of the currently selected test has 
        // updated. If it has then the test details needs to make an api call to get the latest.
        if (this.id == this.listOfTests[i].ID && this.currentTestStatus != status) {
          this.currentTestStatus = status
        }
        if (status == 'Uploading' || status == 'Deploying') {
          tryAgain = true
        }
      }
      if (tryAgain == true) {
        this.getTests()
        setTimeout(this.getTestsLoop, 5000)
        return
      }
      this.gettingTestsLoopActive = false
    }
  }
}
</script>
