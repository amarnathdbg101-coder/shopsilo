package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"shopMe/internal/reuseable"

	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct{
	Id int `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Password string `json:"password"`
}

type RegisterOutput struct{
	Id int `json:"id"`
	Name string `json:"name"`
	Email string `json:"email"`
}

func (uh UserHandler) Register(w http.ResponseWriter, r *http.Request ) {
		// request parse
		// request parse
		var request RegisterRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, "register input data invalid", http.StatusBadRequest)
			return
		}
		// password hashing
		hashed ,err := bcrypt.GenerateFromPassword([]byte(request.Password),10)
		if err != nil{
			http.Error(w,"password hashing error",http.StatusInternalServerError)
			return 
		}

		request.Password = string(hashed)
		// check user
		_,_ ,userEmail,_,err := reuseable.FindByEmail(request.Email,uh.db)
		if err != nil || userEmail != ""{
			http.Error(w,"email already exists",http.StatusBadRequest)
			return 
		}
		// fetch database
		var userData RegisterOutput
		query := `INSERT INTO users(name,email,password) VALUES($1,$2,$3) RETURNING id,name,email`
		row := uh.db.QueryRow(context.Background(),query,request.Name,request.Email,request.Password)
		err = row.Scan(&userData.Id,&userData.Name,&userData.Email)
		if err != nil{
			http.Error(w,"database error",http.StatusInternalServerError)
			return 
		}
	  w.Header().Set("content-type","application/json")
		err = json.NewEncoder(w).Encode(userData)
		if err != nil{
			http.Error(w,"response error",http.StatusInternalServerError)
			return 
		}
    
	}
