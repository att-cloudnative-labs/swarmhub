<template>
  <div class="tile is-7 is-vertical is-parent">
    <div class="tile is-child box">
      <div class="level">
        <div class="level-left">
          <a class="button is-info title is-5" @click="isCreateTestModalActive = true">Create Test</a>
        </div>
        <div class="level-right">
          <div class="tile is-vertical">
            <div class="level">
              <div class="control">
                <p>Start</p>
                <datepicker v-model="startDatePicked"></datepicker>
              </div>
              <div class="control">
                <p>End</p>
                <datepicker v-model="endDatePicked"></datepicker>
              </div>
            </div>
            <div class="field has-addons">
              <div class="control is-expanded">
                <input
                  v-model="textSearch"
                  v-on:keyup.enter="changeFields()"
                  class="input"
                  type="text"
                  placeholder="Text search..."
                />
              </div>
              <p class="control">
                <button class="button is-info" @click="changeFields();">Search</button>
              </p>
            </div>
          </div>
        </div>
      </div>
      <div class="box">
        <div class="table-container">
          <table class="table is-hoverable" style="width:100%">
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Status</th>
              <th>Result</th>
              <th>Labels</th>
              <th>Created</th>
              <th>Launched</th>
              <th>Stopped</th>
            </tr>
            <tr
              v-for="test in listOfTests"
              :key="test.ID"
              v-bind:class="{ 'is-selected': selectedTestID==test.ID }"
              @click="$emit('selected-test', test.ID);"
            >
              <td>..{{ test.ID.substring(test.ID.length - 6, test.ID.length) }}</td>
              <td>{{ test.Name }}</td>
              <td>{{ test.Status }}</td>
              <td>{{ test.Result }}</td>
              <td>
                <p v-for="label in test.Labels" :key="label">{{label}}</p>
              </td>
              <td v-if="test.Created != '-'">{{ test.Created | moment("lll") }}</td>
              <td v-else>{{ test.Created }}</td>
              <td v-if="test.Launched != '-'">{{ test.Launched | moment("lll") }}</td>
              <td v-else>{{ test.Launched }}</td>
              <td v-if="test.Stopped != '-'">{{ test.Stopped | moment("lll") }}</td>
              <td v-else>{{ test.Stopped }}</td>
            </tr>
          </table>
        </div>
      </div>
      <nav class="pagination is-right" role="navigation" aria-label="pagination">
        <button
          v-bind:disabled="firstTestID===firstTestIDInList"
          class="pagination-previous"
          @click="$emit('previous-page', firstTestIDInList);"
        >Previous</button>
        <button
          v-bind:disabled="lastTestID===lastTestIDInList"
          class="pagination-next"
          @click="$emit('next-page', lastTestIDInList);"
        >Next page</button>
      </nav>
      <create-test
        :isCreateTestModalActive="isCreateTestModalActive"
        @is-active="updateCreateTestBool"
        @get-tests="getAllTests"
      />
    </div>
  </div>
</template>

<script>
import CreateTest from "@/components/CreateTest.vue";
import Datepicker from "vuejs-datepicker";

export default {
  name: "TestList",
  components: {
    CreateTest,
    Datepicker
  },
  props: {
    selectedTestID: String,
    listOfTests: Array,
    firstTestIDInList: String,
    lastTestIDInList: String,
    firstTestID: String,
    lastTestID: String
  },
  data: function() {
    return {
      isCreateTestModalActive: false,
      startDatePicked: null,
      endDatePicked: null,
      textSearch: ""
    };
  },
  methods: {
    updateCreateTestBool(bool) {
      this.isCreateTestModalActive = bool;
    },
    getAllTests: function(testID) {
      this.$emit("get-tests", testID);
    },
    changeFields() {
      var startDate = "";
      var endDate = "";
      if (this.startDatePicked != null) {
        this.startDatePicked.setHours(0, 0, 0, 0);
        startDate =
          this.startDatePicked.getFullYear() +
          "-" +
          ("0" + (this.startDatePicked.getMonth() + 1)).slice(-2) +
          "-" +
          ("0" + this.startDatePicked.getDate()).slice(-2) +
          "T" +
          ("0" + this.startDatePicked.getHours()).slice(-2) +
          ":" +
          ("0" + this.startDatePicked.getMinutes()).slice(-2) +
          ":" +
          "00";
      }
      if (this.startDatePicked != null) {
        this.endDatePicked.setHours(23, 59, 59, 999);
        endDate =
          this.endDatePicked.getFullYear() +
          "-" +
          ("0" + (this.endDatePicked.getMonth() + 1)).slice(-2) +
          "-" +
          ("0" + this.endDatePicked.getDate()).slice(-2) +
          "T" +
          ("0" + this.endDatePicked.getHours()).slice(-2) +
          ":" +
          ("0" + this.endDatePicked.getMinutes()).slice(-2) +
          ":" +
          "59";
      }
      this.$emit("get-tests-fields", this.textSearch, startDate, endDate);
    }
  }
};
</script>
