package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
)

type HandlerContext struct {
	db *dal.Query
}

type CreateProductsRequest struct {
	Products *[]model.Product `json:"products"`
}

func (ctx *HandlerContext) InitialHandlerContext(db *dal.Query) {
	ctx.db = db
}

// CreateProducts Create new Products
func (ctx *HandlerContext) CreateProducts(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodPost}, w, r) {
		return
	}

	req := CreateProductsRequest{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate Payload
	validationErr := ""
	for i := range *req.Products {
		if (*req.Products)[i].Name == "" {
			validationErr += fmt.Sprintf("The %d product name is required.", i+1)
		}
	}
	if validationErr != "" {
		http.Error(w, validationErr, http.StatusBadRequest)
	}
	err = ctx.db.Transaction(func(tx *dal.Query) error {
		for _, product := range *req.Products {
			if errCreate := tx.Product.Create(&product); errCreate != nil {
				return errors.New(product.Name + ": " + errCreate.Error())
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Create Products Success"))
}

// SetProductsInventory Set Product inventory
func (ctx *HandlerContext) QueryProduct(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodGet}, w, r) {
		return
	}

	req := model.Product{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate payload
	if req.ID == 0 {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	productInfo, errDb := ctx.db.Product.Where(ctx.db.Product.ID.Eq(req.ID)).First()
	if errDb != nil {
		http.Error(w, errDb.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "application/json")
	respBody, _ := json.Marshal(*productInfo)
	w.Write(respBody)
}
