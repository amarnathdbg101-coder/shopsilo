package reuseable

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetById(id any, db *pgxpool.Pool)(int,string,string,error) {
	var user userData
	query := `select * from users where id=$1`
	err := db.QueryRow(context.Background(),query,id).Scan(&user.Id,&user.Name,&user.Email)
	if err != nil{
		log.Println("user not found")
	}
	return user.Id,user.Name,user.Email,nil
}