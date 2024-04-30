package order

import (
	"encoding/json"
	"errors"
	"github.com/romana/rlog"
	"net/http"
	"order_system/constants"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
)

type PaymentMethod func(*model.Order) error

type HandlerContext struct {
	db            *dal.Query
	orderChan     chan *model.Order
	paymentMethod PaymentMethod
	PaymentMQUrl  string
}

type CreateOrderRequest struct {
	CustomerId   uint    `json:"customer_id"`
	CustomerName *string `json:"customer_name,omitempty"`
	ProductId    uint    `json:"product_id"`
	ProductName  *string `json:"product_name,omitempty"`
}

type PaymentCallBackRequest struct {
	OrderId       uint          `json:"order_id"`
	PaymentDetail model.Payment `json:"payment_detail"`
}

func (ctx *HandlerContext) InitialHandlerContext(db *dal.Query, paymentMethod PaymentMethod, paymentMQUrl string) {
	ctx.db = db
	ctx.paymentMethod = paymentMethod
	ctx.orderChan = make(chan *model.Order, 10000)
	ctx.PaymentMQUrl = paymentMQUrl
}

// CreateOrder Create a new Order
func (ctx *HandlerContext) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodPost}, w, r) {
		return
	}

	req := CreateOrderRequest{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Validate payload
	if req.CustomerId == 0 {
		http.Error(w, "CustomerId is required", http.StatusBadRequest)
		return
	}
	if req.ProductId == 0 {
		http.Error(w, "ProductId is required", http.StatusBadRequest)
		return
	}

	// Save to DB
	newOrder := model.Order{
		CustomerId: req.CustomerId,
		ProductId:  req.ProductId,
		Amount:     100,
		State:      ORDER_STATE_CREATED,
	}
	errDb := ctx.db.Transaction(func(tx *dal.Query) error {
		// Update Product
		result, errTx := tx.Product.Where(tx.Product.ID.Eq(req.ProductId), tx.Product.IsAvailable.Is(true)).Update(tx.Product.IsAvailable, false)
		if errTx != nil || result.RowsAffected == 0 {
			return errors.New(constants.PRODUCT_NOT_AVAILABLE)
		}
		// Create new order
		errTx = tx.Order.Create(&newOrder)
		if errTx != nil {
			errInfo := constants.CREATE_ORDER_FAILED + ": " + errTx.Error()
			return errors.New(errInfo)
		}
		return nil
	})

	if errDb != nil {
		http.Error(w, errDb.Error(), http.StatusInternalServerError)
		return
	}

	// write to order chan
	rlog.Infof("Order was created as state %d(%s)", ORDER_STATE_CREATED, stateCodeToString(ORDER_STATE_CREATED))
	ctx.orderChan <- &newOrder

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("create order success."))
}

// Fetch order detail by order id
func (ctx *HandlerContext) QueryOrder(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodGet}, w, r) {
		return
	}

	req := model.Order{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Validate payload
	if req.ID == 0 {
		http.Error(w, "Order id is required", http.StatusBadRequest)
		return
	}

	orderDetail, errDB := ctx.db.Order.Where(ctx.db.Order.ID.Eq(req.ID)).First()
	if errDB != nil {
		rlog.Error(errDB.Error())
		http.Error(w, errDB.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(*orderDetail)
	w.Write(respBody)

}

// PaymentResultCallBack update order status when payment is done.
func (ctx *HandlerContext) PaymentCallBack(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodPost}, w, r) {
		return
	}

	req := PaymentCallBackRequest{}
	err := util.FetchReqObject(r, &req)
	if err != nil {
		rlog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch Order
	orderInfo, errDB := ctx.db.Order.Where(ctx.db.Order.ID.Eq(req.OrderId)).First()
	if errDB != nil || orderInfo == nil {
		errInfo := "Order not found"
		if errDB != nil {
			errInfo = "Order not found: " + errDB.Error()
		}
		rlog.Error(errInfo)
		http.Error(w, errInfo, http.StatusInternalServerError)
		return
	}
	// Validate order state
	if orderInfo.State != ORDER_STATE_AWAITPAYMENT {
		errInfo := "Order is not AWAIT PAYMENT"
		http.Error(w, errInfo, http.StatusBadRequest)
		return
	}

	updOrderObj := model.Order{}
	newOrderState := ORDER_STATE_PAID
	if req.PaymentDetail.State == constants.PAYMENT_STATE_FAILED {
		newOrderState = ORDER_STATE_FAILED
		updOrderObj.FailReason = req.PaymentDetail.PaymentResult
	}
	updOrderObj.State = newOrderState

	// update order state
	result, errDB := ctx.db.Order.Where(ctx.db.Order.ID.Eq(req.OrderId)).Updates(updOrderObj)
	if errDB != nil || result.RowsAffected == 0 {
		errInfo := "Update order status fail"
		http.Error(w, errInfo, http.StatusInternalServerError)
		return
	}

	// Write to order chan
	orderInfo.State = newOrderState
	rlog.Infof("Order %d state was seted to %d(%s)", orderInfo.ID, orderInfo.State, stateCodeToString(orderInfo.State))
	ctx.orderChan <- orderInfo

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("Update order payment info success."))
}
