package shop

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
)

type Input struct {
	ShopName  string  `json:"name"`
	Timing    string  `json:"timing"`
	Village   string  `json:"village"`
	Post      string  `json:"post"`
	District  string  `json:"district"`
	State     string  `json:"state"`
	Pincode   string  `json:"pincode"`
	Landmark  string  `json:"landmark"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (sh *ShopHandler) CreateShop(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(middleware.UserIDContextKey).(int)
	var input Input
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, `{"status":"error","message":"invalid input"}`, http.StatusBadRequest)
		return
	}

	// insert address
	var addressId int
	query1 := `insert into addresses (village,post,district,state,pincode,landmark,latitude,longitude) values ($1,$2,$3,$4,$5,$6,$7,$8) returning id`
	err = sh.db.QueryRow(r.Context(), query1, input.Village, input.Post, input.District, input.State, input.Pincode, input.Landmark, input.Latitude, input.Longitude).Scan(&addressId)
	if err != nil {
		http.Error(w, `{"status":"error","message":"address insert failed"}`, http.StatusInternalServerError)
		return
	}

	// insert shop
	var shopId int
	query2 := `insert into shops (owner_id,name,address_id,timing) values ($1,$2,$3,$4) returning id`
	err = sh.db.QueryRow(r.Context(), query2, userId, input.ShopName, addressId, input.Timing).Scan(&shopId)
	if err != nil {
		http.Error(w, `{"status":"error","message":"shop insert failed"}`, http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(map[string]any{
		"status":    "success",
		"message":   "shop created successfully",
		"shopId":    shopId,
		"shopName":  input.ShopName,
		"timing":    input.Timing,
		"addressId": addressId,
	})
	if err != nil {
		http.Error(w, `{"status":"error","message":"response encoding failed"}`, http.StatusInternalServerError)
		return
	}
}
