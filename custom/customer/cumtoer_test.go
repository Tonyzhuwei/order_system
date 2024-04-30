package customer

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
	"testing"
)

var (
	testCustomer = model.Customer{
		ID:      1,
		Name:    "Test Customer",
		Email:   util.GetStringPtr("user@mail.com"),
		Address: util.GetStringPtr("this is a test address"),
	}
)

func TestQueryCustomerSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	returnData, _ := util.ObjectToRows(testCustomer)
	expectedSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	mock.ExpectQuery(expectedSQL).WithArgs(testCustomer.ID, 1).WillReturnRows(returnData)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{"customer_id":1}`)))
	handlerCtx.QueryCustomer(w, r)

	actualResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, testCustomer, actualResp, "Unexpected result")
}

func TestQueryCustomerBadHttpMethod(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer([]byte(`{}`)))
	handlerCtx.QueryCustomer(w, r)

	actualResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestQueryCustomerWithoutCustomerID(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{}`)))
	handlerCtx.QueryCustomer(w, r)

	actualResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQueryCustomerNotFound(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	//returnData, _ := util.ObjectToRows(testCustomer)
	expectedSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	mock.ExpectQuery(expectedSQL).WithArgs(testCustomer.ID, 1).WillReturnError(gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{"customer_id":1}`)))
	handlerCtx.QueryCustomer(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreatCustomerSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	creatCustomerSQL := "INSERT INTO \"customers\" .+ VALUES .+"
	newRows, _ := util.ObjectToRows(testCustomer)
	mock.ExpectBegin()
	mock.ExpectQuery(creatCustomerSQL).WillReturnRows(newRows)
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateCustomerRequest{Customers: &[]model.Customer{testCustomer}})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateCustomers(w, r)

	actualResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreatCustomerBadHttpMethod(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateCustomerRequest{Customers: &[]model.Customer{testCustomer}})
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateCustomers(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCreatCustomerMissingName(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	customerWithoutName := testCustomer
	customerWithoutName.Name = ""
	customers := []model.Customer{
		customerWithoutName,
	}
	requestObj := CreateCustomerRequest{
		Customers: &customers,
	}

	reqBody, _ := json.Marshal(requestObj)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateCustomers(w, r)

	acutalResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &acutalResp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatCustomerNameExisting(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	creatCustomerSQL := "INSERT INTO \"customers\" .+ VALUES .+"
	mock.ExpectBegin()
	mock.ExpectQuery(creatCustomerSQL).WillReturnError(gorm.ErrDuplicatedKey)

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateCustomerRequest{Customers: &[]model.Customer{testCustomer}})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateCustomers(w, r)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
