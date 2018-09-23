package response

import (
	"encoding/json"
	"net/http"
)

// WriteErrorResponse adds even more convenience for 400-level errors
func WriteErrorResponse(w http.ResponseWriter, err error) {
	errorData := map[string]string{
		"error": err.Error(),
	}

	Write(w, errorData, http.StatusBadRequest)
}

// Write handles some of the more repetetive elements of writing http responses
func Write(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusBadRequest)

	// For now, we're assuming json.Marshal succeeds...
	marshalledData, _ := json.Marshal(data)
	w.Write(marshalledData)
}
