package customer

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

type CreateCustomerRequest struct {
	Customers *[]model.Customer `json:"customers"`
}

type QueryCustomerRequest struct {
	CustomerId uint `json:"customer_id"`
}

func (ctx *HandlerContext) InitialHandlerContext(db *dal.Query) {
	ctx.db = db
}

// Create new customers
func (ctx *HandlerContext) CreateCustomers(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodPost}, w, r) {
		return
	}

	req := CreateCustomerRequest{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate payload
	validationErr := ""
	for i := range *req.Customers {
		if (*req.Customers)[i].Name == "" {
			validationErr += fmt.Sprintf("The %d customer name is required.", i+1)
		}
	}
	if validationErr != "" {
		http.Error(w, validationErr, http.StatusBadRequest)
	}

	err = ctx.db.Transaction(func(tx *dal.Query) error {
		for _, customer := range *req.Customers {
			if errCreate := tx.Customer.Create(&customer); errCreate != nil {
				return errors.New(customer.Name + " : " + errCreate.Error())
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Create Customer Success"))
}

// Query customer
func (ctx *HandlerContext) QueryCustomer(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodGet}, w, r) {
		return
	}

	req := QueryCustomerRequest{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate payload
	if req.CustomerId == 0 {
		http.Error(w, "CustomerId is required", http.StatusBadRequest)
		return
	}

	customerInfo, errQuery := ctx.db.Customer.Where(dal.Customer.ID.Eq(req.CustomerId)).First()

	if errQuery != nil {
		http.Error(w, errQuery.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(*customerInfo)
	w.Write(respBody)
}
