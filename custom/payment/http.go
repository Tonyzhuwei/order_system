package payment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/romana/rlog"
	"net/http"
	"order_system/constants"
	"order_system/custom/message_queue"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
)

type PaymentMethod func(*model.Order) error
type OrderCallBackMethod func(PaymentCallBackRequest) error

type HandlerContext struct {
	db                  *dal.Query
	mq                  *message_queue.MessageQueue
	paymentMethod       PaymentMethod
	OrderCallBackUrl    string
	OrderCallbackMethod OrderCallBackMethod
}

type PaymentCallBackRequest struct {
	OrderId       uint          `json:"order_id"`
	PaymentDetail model.Payment `json:"payment_detail"`
}

func (ctx *HandlerContext) InitialHandlerContext(db *dal.Query, mq *message_queue.MessageQueue, payMethod PaymentMethod, callBackUrl string, orderCallbackMethod OrderCallBackMethod) {
	ctx.db = db
	ctx.mq = mq
	ctx.paymentMethod = payMethod
	ctx.OrderCallBackUrl = callBackUrl
	ctx.OrderCallbackMethod = orderCallbackMethod
}

// PublishPaymentMQ receive payment request from Order system and push it to MQ
func (ctx *HandlerContext) PublishPaymentMQ(w http.ResponseWriter, r *http.Request) {
	// Validate http method
	if !util.IsAllowHttpMethod([]string{http.MethodPost}, w, r) {
		return
	}

	orderInfo := model.Order{}
	err := util.FetchReqObject(r, &orderInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//Validate Payload
	validationErrs := make([]error, 0)
	if orderInfo.ID <= 0 {
		validationErrs = append(validationErrs, errors.New("Order ID is invalid"))
	}
	if orderInfo.CustomerId <= 0 {
		validationErrs = append(validationErrs, errors.New("Customer ID is invalid"))
	}
	if orderInfo.Amount < 0 {
		validationErrs = append(validationErrs, errors.New("Order Amount is invalid"))
	}
	if len(validationErrs) > 0 {
		errInfo := ""
		for i := range validationErrs {
			errInfo = errInfo + "\n" + validationErrs[i].Error()
		}
		http.Error(w, errInfo, http.StatusBadRequest)
		return
	}

	rlog.Infof("Got a new payment, OrderId=%d, Amout=%f", orderInfo.ID, orderInfo.Amount)
	ctx.mq.Enqueue(&orderInfo)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Operation success."))
	return
}

// ConsumePaymentMQ Consume message from MQ and start new payment
func (ctx *HandlerContext) ConsumePaymentMQ() {
	for {
		newPaymentOrder := ctx.mq.Dequeue()
		if newPaymentOrder == nil {
			continue
		}
		go ctx.startNewPayment(newPaymentOrder)
	}
}

// Starting a new payment
func (ctx *HandlerContext) startNewPayment(newOrder *model.Order) {
	rlog.Info("Starting process payment.")
	err := ctx.paymentMethod(newOrder)
	// Set payment failed
	if err != nil {
		rlog.Info("Process payment failed.")

	} else {
		rlog.Info("Payment completed.")

	}
	// Notify Order system
	rlog.Info("Notify order system")
	ctx.notifyOrderSystem(&model.Payment{
		OrderId: newOrder.ID,
		State:   constants.PAYMENT_STATE_SUCCESS,
	})
}

// ProcessPaymentMethod Process payment, will be mocked in unit test cases
func (ctx *HandlerContext) ProcessPaymentMethod(newOrder *model.Order) error {
	// create new payment

	// start operation

	return nil
}

// Notify payment result to Order system
func (ctx *HandlerContext) notifyOrderSystem(payment *model.Payment) {
	reqObj := PaymentCallBackRequest{
		OrderId:       payment.OrderId,
		PaymentDetail: *payment,
	}

	err := ctx.OrderCallbackMethod(reqObj)
	if err != nil {
		rlog.Error("Call order payment call back API failed with err: ", err.Error())
	}
	return
}

// CallPaymentCallbackAPI call order system's paymentCallback api, will be mocked in unit test cases
func (ctx *HandlerContext) CallPaymentCallbackAPI(reqObj PaymentCallBackRequest) error {
	reqBody, err := json.Marshal(reqObj)
	r, err := http.NewRequest(http.MethodPost, ctx.OrderCallBackUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		rlog.Error(err)
		return err
	}
	r.Header.Add("Content-Type", "application/json")
	reponse, err := http.DefaultClient.Do(r)
	if err != nil {
		rlog.Error(err)
		return err
	}
	if reponse.StatusCode != http.StatusOK {
		errInfo := fmt.Sprintf("Notify Order system failed with Status code %d", reponse.StatusCode)
		rlog.Errorf(errInfo)
		return errors.New(errInfo)
	}
	return nil
}
