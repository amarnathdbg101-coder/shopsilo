package reuseable

import (
	"context"
	"log"


	"github.com/jackc/pgx/v5/pgxpool"
)

type userData struct{
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func FindByEmail(email string, db *pgxpool.Pool) (int,string,string,string,error) {
	var user userData
	query := `select id,name, email,password from users where email=$1`
	err := db.QueryRow(context.Background(), query, email).Scan(&user.Id,&user.Name,&user.Email,&user.Password)
	if err != nil {
		log.Print("user not found")
	}
	return user.Id,user.Name,user.Email,user.Password,nil
}
