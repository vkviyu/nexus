package response

import (
	"encoding/json"
	"net/http"
)

func WriteJSONResponse(w http.ResponseWriter, body interface{}, statusCode int) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}

func WriteRawJSONResponse(w http.ResponseWriter, body []byte, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(body)
}

func WriteOK(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusOK)
}

func WriteOKRawJSON(w http.ResponseWriter, body []byte) {
	WriteRawJSONResponse(w, body, http.StatusOK)
}

func WriteBadRequest(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusBadRequest)
}

func WriteUnauthorized(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusUnauthorized)
}

func WriteForbidden(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusForbidden)
}

func WriteNotFound(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusNotFound)
}



func WriteConflict(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusConflict)
}

func WriteInternalServerError(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusInternalServerError)
}

func WriteBadGateway(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusBadGateway)
}

func WriteGatewayTimeout(w http.ResponseWriter, body interface{}) {
	WriteJSONResponse(w, body, http.StatusGatewayTimeout)
}