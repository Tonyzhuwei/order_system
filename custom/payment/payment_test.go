package payment

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"net/http"
	"net/http/httptest"
	"order_system/constants"
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
	testPayment = model.Payment{
		ID:              1,
		OrderId:         1,
		Amount:          100.00,
		State:           constants.PAYMENT_STATE_CREATED,
		IsNotifiedOrder: false,
	}
)

func mockProcessPayment(order *model.Order) error {
	if order.Amount > 1000 {
		return errors.New("exceed payment limit")
	}
	return nil
}

func mockPaymentCallBackAPI(request PaymentCallBackRequest) error {
	return nil
}

func TestPublishPaymentMQSuccess(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

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
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

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
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

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

func TestStartNewPaymentSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

	expectSql := ".+"
	rows, _ := util.ObjectToRows(testPayment)
	mock.ExpectBegin()
	mock.ExpectQuery(expectSql).WillReturnRows(rows)
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(expectSql).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := handlerCtx.startNewPayment(&testOrder)
	assert.Nil(t, err)
}

func TestStartNewPaymentInvalidOrder(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

	newOrder := testOrder
	newOrder.ID = 0
	err := handlerCtx.startNewPayment(&newOrder)
	assert.Error(t, err)

	newOrder = testOrder
	newOrder.Amount = -100.00
	err = handlerCtx.startNewPayment(&newOrder)
	assert.Error(t, err)
}

func TestStartNewPaymentExceedLimit(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

	expectSql := ".+"
	rows, _ := util.ObjectToRows(testPayment)
	mock.ExpectBegin()
	mock.ExpectQuery(expectSql).WillReturnRows(rows)
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(expectSql).WithArgs(
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		constants.PAYMENT_STATE_FAILED,
		sqlmock.AnyArg(),
		true,
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	newOrder := testOrder
	newOrder.Amount = 2000.00
	err := handlerCtx.startNewPayment(&newOrder)
	assert.Error(t, err)
	assert.Equal(t, constants.EXCEED_PAYMENT_LIMIT, err.Error())
}

func TestNotifyOrderSystemSuccess(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

	newPayment := testPayment
	newPayment.State = constants.PAYMENT_STATE_SUCCESS
	err := handlerCtx.notifyOrderSystem(&newPayment)
	assert.Nil(t, err)
}

func TestNotifyOrderSystemFail(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	mq := message_queue.NewMessageQueue()
	handlerCtx.InitialHandlerContext(dal.Q, mq, mockProcessPayment, "", mockPaymentCallBackAPI)

	err := handlerCtx.notifyOrderSystem(nil)
	assert.Error(t, err)
}
