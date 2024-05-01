package order

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/romana/rlog"
	"net/http"
	"order_system/custom/util"
	"order_system/model"
)

// Order States
const ORDER_STATE_CREATED = int8(0)
const ORDER_STATE_AWAITPAYMENT = int8(1)
const ORDER_STATE_PAID = int8(2)
const ORDER_STATE_FULFILLED = int8(3)
const ORDER_STATE_FAILED = int8(4)
const ORDER_STATE_CANCELED = int8(5)

func stateCodeToString(state int8) string {
	switch state {
	case ORDER_STATE_CREATED:
		return "CREATED"
	case ORDER_STATE_AWAITPAYMENT:
		return "AWAIT PAYMENT"
	case ORDER_STATE_PAID:
		return "PAID"
	case ORDER_STATE_FULFILLED:
		return "FULFILLED"
	case ORDER_STATE_FAILED:
		return "FAILED"
	case ORDER_STATE_CANCELED:
		return "CANCELED"
	}
	return "UNKNOWN"
}

// ScanPendingOrders Will be used to fetch pending orders and trigger them agan when starting
func (ctx *HandlerContext) ScanPendingOrders() {
	orderTable := ctx.db.Order
	pendingOrders, err := orderTable.Where(orderTable.State.In(ORDER_STATE_CREATED, ORDER_STATE_PAID)).Find()
	if err != nil {
		rlog.Error(err)
		return
	}
	if len(pendingOrders) > 0 {
		rlog.Infof("Found %d pending orders.", len(pendingOrders))
	}
	for _, order := range pendingOrders {
		ctx.orderChan <- order
	}
}

// ExecuteOrders Execute orders in background go routines
func (ctx *HandlerContext) ExecuteOrders() {
	for true {
		orderDetail := <-ctx.orderChan
		if orderDetail == nil {
			continue
		}
		go func() {
			switch orderDetail.State {
			case ORDER_STATE_CREATED:
				ctx.makePayment(orderDetail)
			case ORDER_STATE_PAID:
				ctx.fulfillOrder(orderDetail)
			}
		}()
	}
}

// Make a payment
func (ctx *HandlerContext) makePayment(order *model.Order) error {
	rlog.Info("Calling payment async API....")
	errCallPayment := ctx.paymentMethod(order)
	if errCallPayment != nil {
		rlog.Error("Call payment fail:", errCallPayment.Error())
		// Retry
		errRetry := ctx.paymentMethod(order)
		if errRetry != nil {
			rlog.Error("Retry call payment error:", errRetry.Error())
			_, err := ctx.db.Order.Where(ctx.db.Order.ID.Eq(order.ID)).Updates(model.Order{FailReason: util.GetStringPtr("Failed to call payment api")})
			if err != nil {
				rlog.Error("Update order fail:", err.Error())
			}
			return errRetry
		}
	}
	rlog.Info("Call payment complete")
	_, err := ctx.db.Order.Where(ctx.db.Order.ID.Eq(order.ID)).Updates(model.Order{State: ORDER_STATE_AWAITPAYMENT})
	if err != nil {
		rlog.Error(err)
		return err
	}
	rlog.Infof("Order %d state was seted to %d(%s)", order.ID, ORDER_STATE_AWAITPAYMENT, stateCodeToString(ORDER_STATE_AWAITPAYMENT))
	return nil
}

// Fulfill the order
func (ctx *HandlerContext) fulfillOrder(order *model.Order) {
	// Assume always success
	rlog.Info("Processing order...")

	_, err := ctx.db.Order.Where(ctx.db.Order.ID.Eq(order.ID)).Updates(model.Order{State: ORDER_STATE_FULFILLED})
	if err != nil {
		rlog.Error(err)
	}
	rlog.Infof("Order %d state was seted to %d(%s)", order.ID, ORDER_STATE_FULFILLED, stateCodeToString(ORDER_STATE_FULFILLED))
}

// CallPaymentApi method for Notifying payment API to start a new payment
func (ctx *HandlerContext) CallPaymentApi(order *model.Order) error {
	reqBody, err := json.Marshal(*order)
	if err != nil {
		rlog.Error(err)
		return err
	}
	r, err := http.NewRequest(http.MethodPost, ctx.PaymentMQUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		rlog.Error(err)
		return err
	}
	r.Header.Add("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		rlog.Error(err)
		return err
	}
	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Notify order failed with status code %d", response.StatusCode))
	}
	return nil
}
