package reuseable

import (
	"shopMe/internal/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJwt(userId int, email string) (string, error) {
	// Payload
	claims := jwt.MapClaims{
		"userId":userId,
		"email":email,
		"exp": time.Now().Add(time.Hour*720).Unix(),
	}
	token :=  jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	return token.SignedString([]byte(utils.MustLoad().Jwt))
}