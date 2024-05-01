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
	"strings"
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

	rlog.Infof("Got a new payment, OrderId=%d, Amout=%.2f", orderInfo.ID, orderInfo.Amount)
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
func (ctx *HandlerContext) startNewPayment(newOrder *model.Order) error {
	// Validate Order
	if newOrder.ID <= 0 {
		return errors.New(fmt.Sprintf("Order ID [%d] is invalid", newOrder.ID))
	}
	if newOrder.Amount < 0 {
		return errors.New(fmt.Sprintf("Order Amount [%.2f] is invalid", newOrder.Amount))
	}
	// Create new payment
	newPayment := model.Payment{
		OrderId:         newOrder.ID,
		Amount:          newOrder.Amount,
		State:           constants.PAYMENT_STATE_CREATED,
		IsNotifiedOrder: false,
	}
	paymentTable := ctx.db.Payment
	errDb := paymentTable.Create(&newPayment)
	if errDb != nil {
		return errors.New("Failed to create payment in DB with Error: " + errDb.Error())
	}
	rlog.Infof("Payment was created, ID=%d,OrderId=%d,Amount=%.2f", newPayment.ID, newPayment.OrderId, newPayment.Amount)

	errArray := make([]string, 0)
	// Process Payment
	rlog.Info("Starting process payment.")
	err := ctx.paymentMethod(newOrder)
	if err != nil {
		errInfo := err.Error()
		errArray = append(errArray, errInfo)
		newPayment.State = constants.PAYMENT_STATE_FAILED
		newPayment.PaymentResult = &errInfo
		rlog.Error("Process payment failed: " + errInfo)
	} else {
		newPayment.State = constants.PAYMENT_STATE_SUCCESS
		newPayment.PaymentResult = util.GetStringPtr("Succeed")
	}

	// Notify Order system
	err = ctx.notifyOrderSystem(&model.Payment{
		OrderId: newOrder.ID,
		State:   newPayment.State,
	})
	newPayment.IsNotifiedOrder = true
	if err != nil {
		newPayment.IsNotifiedOrder = false
		rlog.Errorf("Notify Payment(PaymentId=%d,OrderId=%d) result to Order System failed due to: %s", newPayment.ID, newPayment.OrderId, err.Error())
		errArray = append(errArray, err.Error())
	}

	// Update payment result to DB
	updateResult, err := paymentTable.Where(paymentTable.ID.Eq(newPayment.ID)).Updates(newPayment)
	if err != nil || updateResult.RowsAffected == 0 {
		errInfo := "Update payment state failed"
		if err != nil {
			errInfo += " with error: " + err.Error()
		}
		errArray = append(errArray, errInfo)
		rlog.Error(errInfo)
	} else {
		rlog.Infof("Payment(ID=%d) state was update to %d", newPayment.ID, newPayment.State)
	}

	if len(errArray) > 0 {
		errInfo := strings.Join(errArray, "\n")
		return errors.New(errInfo)
	}
	return nil

}

// ProcessPaymentMethod Process payment, will be mocked in unit test cases
func (ctx *HandlerContext) ProcessPaymentMethod(newOrder *model.Order) error {
	// Call bank or 3rd party payment service to process payment.
	// Assume always success, only failure when exceed payment limit
	if newOrder.Amount > 1000 {
		return errors.New(constants.EXCEED_PAYMENT_LIMIT)
	}

	return nil
}

// Notify payment result to Order system
func (ctx *HandlerContext) notifyOrderSystem(payment *model.Payment) error {
	if payment == nil {
		return errors.New("Payment cannot be nil.")
	}

	reqObj := PaymentCallBackRequest{
		OrderId:       payment.OrderId,
		PaymentDetail: *payment,
	}

	err := ctx.OrderCallbackMethod(reqObj)
	if err != nil {
		rlog.Error("Call Order payment callback API failed with err: ", err.Error())
	} else {
		rlog.Infof("Call Order Payment callback API succeed.")
	}

	return err
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
