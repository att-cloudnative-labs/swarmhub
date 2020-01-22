package api

import (
	"encoding/json"
	"fmt"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func CreateGridTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var gridTemplate db.GridTemplate

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&gridTemplate)
	if err != nil {
		message := fmt.Sprintf("invalid grid template data")
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	gridTemplate, err = db.CreateGridTemplate(gridTemplate)
	if err != nil {
		message := fmt.Sprintf("error creating grid template: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(gridTemplate)
	if err != nil {
		message := fmt.Sprintf("error gettig grid template result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(json)
}

func GetAllGridTemplates(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	gridTemplates, err := db.GetAllGridTemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(gridTemplates) == 0 {
		http.Error(w, "No content", http.StatusNoContent)
		return
	}

	json, err := json.Marshal(gridTemplates)
	if err != nil {
		message := fmt.Sprintf("error parsing grid templates result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(json)
}

func GetGridTemplateById(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	gridTemplate, err := db.GetGridTemplateById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(gridTemplate)
	if err != nil {
		message := fmt.Sprintf("error parsing grid templates result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(json)
}

func UpdateGridTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	var gridTemplate db.GridTemplate

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&gridTemplate)
	if err != nil {
		message := fmt.Sprintf("invalid grid template data")
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	err = db.UpdateGridTemplate(id, gridTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("grid template updated"))
}

func DeleteGridTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	err := db.DeleteGridTemplate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("grid template deleted"))
}
