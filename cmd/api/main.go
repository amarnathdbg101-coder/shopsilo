package main

import (
	"net/http"
	"shopMe/internal/routes"
	"shopMe/internal/utils"
)



func main() {

	db := utils.ConnectDB(utils.MustLoad().DatabaseUrl)

	routes := routes.RouteSetup(db)
   
    if err:= http.ListenAndServe(":" + utils.MustLoad().Port, routes); err != nil{
		return
	}
}