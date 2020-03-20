<template>
  <div class="modal" v-bind:class="{ 'is-active': isCreateTestModalActive}">
    <div class="modal-background"></div>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Create Test</p>
        <button
          class="delete"
          aria-label="close"
          @click="$emit('is-active', false); clearCreateTest();"
        ></button>
      </header>
      <section class="modal-card-body">
        <div class="field">
          <label class="label">Name</label>
          <div class="control">
            <input
              class="input"
              v-model="createTestData.Name"
              type="text"
              placeholder="Name of the Test"
            />
          </div>
        </div>
        <div class="field">
          <label class="label">Description</label>
          <div class="control">
            <textarea class="textarea" v-model="createTestData.Desc" placeholder="Textarea"></textarea>
          </div>
        </div>
        <div class="file">
          <label class="file-label">
            <input
              class="file-input"
              type="file"
              name="testfile"
              id="fileid"
              accept=".zip"
              v-on:change="handleFileUpload()"
            />
            <span class="file-cta">
              <span class="file-icon">
                <i class="fas fa-upload"></i>
              </span>
              <span class="file-label">Upload a test…</span>
            </span>
            <span class="file-name">{{ filename }}</span>
          </label>
          <progress v-if="uploadPercentage > 0" max="100" :value.prop="uploadPercentage"></progress>
        </div>

        <article v-if="response.Status == 'Failed'" class="message is-danger">
          <div class="message-body">{{ response.Description }}</div>
        </article>
      </section>
      <footer class="modal-card-foot">
        <button type="submit" class="button is-success" @click="createTest();">Submit</button>
        <button class="button" @click="$emit('is-active', false); clearCreateTest();">Cancel</button>
      </footer>
    </div>
  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "CreateTest",
  props: {
    isCreateTestModalActive: Boolean
  },
  data: function() {
    return {
      createTestData: {
        Name: "",
        Desc: ""
      },
      uploadPercentage: 0,
      response: {
        Status: "",
        Description: ""
      },
      file: null,
      filename: ""
    };
  },
  methods: {
    handleFileUpload: function() {
      this.uploadPercentage = 0;
      this.response.Status = "";
      this.response.Description = "";
      this.file = document.getElementById("fileid").files[0];
      this.filename = document.getElementById("fileid").files[0].name;
    },
    createTest: function() {
      var formData;
      formData = new FormData();
      formData.append("file", this.file);
      formData.append("metadata", JSON.stringify(this.createTestData));
      axios
        .post("/api/test", formData, {
          headers: { "Content-Type": "multipart/form-data" },
          validateStatus: function() {
            return true;
          },
          onUploadProgress: function(progressEvent) {
            this.uploadPercentage = parseInt(
              Math.round((progressEvent.loaded * 100) / progressEvent.total)
            );
          }.bind(this)
        })
        .then(response => {
          this.response = response.data;
          if (response.data.Status == "Success") {
            this.$emit("get-tests", "latest");
            this.$emit("is-active", false);
            this.clearCreateTest();
          }
        })
        .catch(error => {
          console.log("FAILURE: ", error);
        });
    },
    clearCreateTest: function() {
      this.createTestData.Name = "";
      this.createTestData.Desc = "";
      this.response.Status = "";
      this.response.Description = "";
      this.uploadPercentage = 0;
      this.file = null;
      this.filename = "";
    }
  }
};
</script>
