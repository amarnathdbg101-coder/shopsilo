package main

import (
	"net/http"
	"shopMe/internal/routes"
	"shopMe/internal/utils"
)



func main() {

	db := utils.ConnectDB(utils.MustLoad().DatabaseUrl)

	routes := routes.RouteSetup(db)
   
    if err:= http.ListenAndServe(":8080", routes); err != nil{
		return
	}
}