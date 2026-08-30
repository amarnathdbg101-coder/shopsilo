package handler

import "net/http"

func Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type","application/json")
	_,err := w.Write([]byte("Product added successfully"))
	if err != nil {
		http.Error(w,"response error",http.StatusBadRequest)
		return
	}
}