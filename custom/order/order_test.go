package order

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"order_system/constants"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
	"strings"
	"testing"
)

var (
	testOrder = model.Order{
		ID:         1,
		CustomerId: 2,
		ProductId:  3,
		Amount:     100.00,
		State:      ORDER_STATE_CREATED,
		FailReason: nil,
	}
	testCustomer = model.Customer{
		ID:      1,
		Name:    "Test Customer",
		Email:   util.GetStringPtr("user@mail.com"),
		Address: util.GetStringPtr("this is a test address"),
	}
)

func mockPayment(order *model.Order) error {
	if order.Amount > 1000 {
		return errors.New("exceed payment limit")
	}
	return nil
}

func TestCallPaymentSuccess(t *testing.T) {
	db, _, mock := util.DbMock(t)
	defer db.Close()
	orderCtx := HandlerContext{}
	orderCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	testOrder := model.Order{
		ID:         1,
		CustomerId: 1,
		ProductId:  1,
		State:      ORDER_STATE_CREATED,
		Amount:     1000,
	}
	updOrderSQL := "UPDATE \"orders\" SET .+"
	mock.ExpectBegin()
	mock.ExpectExec(updOrderSQL).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := orderCtx.makePayment(&testOrder)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Nil(t, err)
}

func TestCallPaymentFail(t *testing.T) {
	db, _, mock := util.DbMock(t)
	defer db.Close()

	orderCtx := HandlerContext{}
	orderCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	testOrder := model.Order{
		ID:         1,
		CustomerId: 1,
		ProductId:  1,
		State:      ORDER_STATE_CREATED,
		Amount:     1001,
	}
	orderCtx.makePayment(&testOrder)

	assert.Nil(t, mock.ExpectationsWereMet())
	//assert.Error(t, err)
}

func TestFulfillOrder(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	updateProductSQL := "UPDATE \"orders\" SET .+"
	mock.ExpectBegin()
	mock.ExpectExec(updateProductSQL).WithArgs(ORDER_STATE_FULFILLED, sqlmock.AnyArg(), testOrder.ID).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	newOrder := testOrder
	newOrder.State = ORDER_STATE_PAID
	handlerCtx.fulfillOrder(&newOrder)

	assert.Nil(t, mock.ExpectationsWereMet())
}

func TestQueryOrderSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	returnData, _ := util.ObjectToRows(testOrder)
	expectedSQL := `^SELECT \* FROM \"orders\" WHERE \"orders\"\.\"id\" \= .* .* LIMIT .*`
	mock.ExpectQuery(expectedSQL).WithArgs(testOrder.ID, 1).WillReturnRows(returnData)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{"id":1}`)))
	handlerCtx.QueryOrder(w, r)

	acutalResp := model.Order{}
	json.Unmarshal(w.Body.Bytes(), &acutalResp)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, testOrder, acutalResp, "Unexpected result")
}

func TestQueryOrderBadHttpMethod(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer([]byte(`{}`)))
	handlerCtx.QueryOrder(w, r)

	actualResp := model.Product{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestQueryOrderWithoutOrderID(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{}`)))
	handlerCtx.QueryOrder(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQueryOrderNotFound(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	//returnData, _ := util.ObjectToRows(testOrder)
	expectedSQL := `^SELECT \* FROM \"orders\" WHERE \"orders\"\.\"id\" \= .* .* LIMIT .*`
	mock.ExpectQuery(expectedSQL).WithArgs(testOrder.ID, 1).WillReturnError(gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{"id":1}`)))
	handlerCtx.QueryOrder(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreatOrderSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	selectCustomerSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	updateProductSQL := "UPDATE \"products\" SET .+"
	creatSQL := "INSERT INTO \"orders\" .+ VALUES .+"
	orderRows, _ := util.ObjectToRows(testOrder)
	customerRows, _ := util.ObjectToRows(testCustomer)
	mock.ExpectBegin()
	mock.ExpectQuery(selectCustomerSQL).WithArgs(testOrder.CustomerId, 1).WillReturnRows(customerRows)
	mock.ExpectQuery(updateProductSQL).
		WithArgs(false, sqlmock.AnyArg(), testOrder.ProductId, true).
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(driver.Value(100.00)))
	mock.ExpectQuery(creatSQL).WillReturnRows(orderRows)
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateOrderRequest{
		CustomerId: testOrder.CustomerId,
		ProductId:  testOrder.ProductId,
	})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreatOrderBadHttpMethod(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateOrderRequest{})
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCreatOrderMissingCustomer(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	requestObj := CreateOrderRequest{
		CustomerId: 0,
	}

	reqBody, _ := json.Marshal(requestObj)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	acutalResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &acutalResp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatOrderMissingProduct(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	requestObj := CreateOrderRequest{
		CustomerId: 1,
		ProductId:  0,
	}

	reqBody, _ := json.Marshal(requestObj)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	acutalResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &acutalResp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatOrderProductCustomerNotFound(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	selectCustomerSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	updateProductSQL := "UPDATE \"products\" SET .+"
	creatSQL := "INSERT INTO \"orders\" .+ VALUES .+"
	orderRows, _ := util.ObjectToRows(testOrder)
	//customerRows, _ := util.ObjectToRows(testCustomer)
	mock.ExpectBegin()
	mock.ExpectQuery(selectCustomerSQL).WithArgs(testOrder.CustomerId, 1).WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(updateProductSQL).WithArgs(false, sqlmock.AnyArg(), testOrder.ProductId, true).WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectQuery(creatSQL).WillReturnRows(orderRows)
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateOrderRequest{
		CustomerId: testOrder.CustomerId,
		ProductId:  testOrder.ProductId,
	})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, constants.CUSTOMER_NOT_FOUND, strings.TrimSpace(w.Body.String()))
}

func TestCreatOrderProductNotAvailable(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	selectCustomerSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	updateProductSQL := "UPDATE \"products\" SET .+"
	creatSQL := "INSERT INTO \"orders\" .+ VALUES .+"
	orderRows, _ := util.ObjectToRows(testOrder)
	customerRows, _ := util.ObjectToRows(testCustomer)
	mock.ExpectBegin()
	mock.ExpectQuery(selectCustomerSQL).WithArgs(testOrder.CustomerId, 1).WillReturnRows(customerRows)
	mock.ExpectQuery(updateProductSQL).
		WithArgs(false, sqlmock.AnyArg(), testOrder.ProductId, true).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectQuery(creatSQL).WillReturnRows(orderRows)
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateOrderRequest{
		CustomerId: testOrder.CustomerId,
		ProductId:  testOrder.ProductId,
	})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, constants.PRODUCT_NOT_AVAILABLE, strings.TrimSpace(w.Body.String()))
}

func TestCreatOrderInsertFailure(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q, mockPayment, "")

	selectCustomerSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	updateProductSQL := "UPDATE \"products\" SET .+"
	creatSQL := "INSERT INTO \"orders\" .+ VALUES .+"
	//orderRows, _ := util.ObjectToRows(testOrder)
	customerRows, _ := util.ObjectToRows(testCustomer)
	mock.ExpectBegin()
	mock.ExpectQuery(selectCustomerSQL).WithArgs(testOrder.CustomerId, 1).WillReturnRows(customerRows)
	mock.ExpectQuery(updateProductSQL).
		WithArgs(false, sqlmock.AnyArg(), testOrder.ProductId, true).
		WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(driver.Value(100.00)))
	mock.ExpectQuery(creatSQL).WillReturnError(gorm.ErrInvalidDB)
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateOrderRequest{
		CustomerId: testOrder.CustomerId,
		ProductId:  testOrder.ProductId,
	})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateOrder(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Regexp(t, constants.CREATE_ORDER_FAILED, strings.TrimSpace(w.Body.String()))
}
