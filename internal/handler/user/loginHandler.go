package handler

import (
	"encoding/json"
	
	"net/http"
	"shopMe/internal/reuseable"
	"shopMe/internal/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserHandler struct{
	db *pgxpool.Pool
}

func NewUserHandler(db *pgxpool.Pool)*UserHandler{
	return &UserHandler{
		db:db ,
	}
}

func (uh UserHandler) Login(w http.ResponseWriter, r *http.Request) {
    // request parse
    var request LoginRequest
    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        http.Error(w, "register input data invalid", http.StatusBadRequest)
        return
    }

    // fetch database
    userID, userName, userEmail, userPass, err := reuseable.FindByEmail(request.Email, uh.db)
    if err != nil || userEmail == "" {
        http.Error(w, "Wrong EmailId", http.StatusBadRequest)
        return
    }

    // check userPassword
    if err := bcrypt.CompareHashAndPassword([]byte(userPass), []byte(request.Password)); err != nil {
        http.Error(w, "Wrong password", http.StatusBadRequest)
        return
    }

    // jwt token
    claims := jwt.MapClaims{
        "userId": userID,              // 👈 important
        "name":   userName,
        "email":  userEmail,
        "exp":    time.Now().Add(time.Hour *720).Unix(),
    }

    tokens := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signedToken, err := tokens.SignedString([]byte(utils.MustLoad().JwtSecret))
    if err != nil {
        http.Error(w, "token generation failed", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json") // 👈 fixed typo
    err = json.NewEncoder(w).Encode(map[string]string{"token": signedToken})
    if err != nil {
        http.Error(w, "response error", http.StatusInternalServerError)
        return
    }
}

