package reuseable

import (
	"encoding/json"
	"net/http"
)

type code string

type errors struct{
	Error errorPayload `json:"error"`
}
type errorPayload struct {
	Code code `json:"code"`
	Message string `json:"message"`
}

func Error(w http.ResponseWriter,status int,message string,code code){
	w.Header().Set("content-type","application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errors{
		errorPayload{
			Code: code,
			Message: message,
		},
	})
}