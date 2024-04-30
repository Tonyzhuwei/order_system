package payment

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"order_system/custom/message_queue"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
	"testing"
)

var (
	testOrder = model.Order{
		ID:         1,
		CustomerId: 2,
		ProductId:  3,
		Amount:     100.00,
		State:      1,
		FailReason: nil,
	}
)

func mockPayment(order *model.Order) error {
	if order.Amount > 1000 {
		return errors.New("exceed payment limit")
	}
	return nil
}

func mockOrderCallAPI(request PaymentCallBackRequest) error {
	return nil
}

func TestPublishPaymentMQSuccess(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockPayment, "", mockOrderCallAPI)

	w := httptest.NewRecorder()
	newOrder := testOrder
	reqBody, _ := json.Marshal(newOrder)
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.PublishPaymentMQ(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mq.GetMsgCount())
}

func TestPublishPaymentMQBadHttpMethod(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockPayment, "", mockOrderCallAPI)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{}`)))
	handlerCtx.PublishPaymentMQ(w, r)

	actualResp := model.Product{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPublishPaymentMQInvalidPayload(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockPayment, "", mockOrderCallAPI)

	w := httptest.NewRecorder()
	// Invalid Customer ID
	newOrder := testOrder
	newOrder.CustomerId = 0
	reqBody, _ := json.Marshal(newOrder)
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.PublishPaymentMQ(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Invalid Product id
	newOrder = testOrder
	newOrder.CustomerId = 0
	reqBody, _ = json.Marshal(newOrder)
	r = httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.PublishPaymentMQ(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Invalid Order Amount
	newOrder = testOrder
	newOrder.Amount = -10.00
	reqBody, _ = json.Marshal(newOrder)
	r = httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.PublishPaymentMQ(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
