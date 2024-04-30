package product

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
	testProduct = model.Product{
		ID:          1,
		Name:        "test product",
		Description: util.GetStringPtr("this is a test product"),
		Price:       100.00,
		IsAvailable: true,
	}
)

func TestQueryProductSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	returnData, _ := util.ObjectToRows(testProduct)
	expectedSQL := `^SELECT \* FROM \"products\" WHERE \"products\"\.\"id\" \= .* .* LIMIT .*`
	mock.ExpectQuery(expectedSQL).WithArgs(testProduct.ID, 1).WillReturnRows(returnData)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{"id":1}`)))
	handlerCtx.QueryProduct(w, r)

	acutalResp := model.Product{}
	json.Unmarshal(w.Body.Bytes(), &acutalResp)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, testProduct, acutalResp, "Unexpected result")
}

func TestQueryCustomerBadHttpMethod(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer([]byte(`{}`)))
	handlerCtx.QueryProduct(w, r)

	actualResp := model.Product{}
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
	handlerCtx.QueryProduct(w, r)

	actualResp := model.Product{}
	json.Unmarshal(w.Body.Bytes(), &actualResp)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQueryCustomerNotFound(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	expectedSQL := `^SELECT \* FROM \"customers\" WHERE \"customers\"\.\"id\" \= .* .* LIMIT .*`
	mock.ExpectQuery(expectedSQL).WithArgs(testProduct.ID, 1).WillReturnError(gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer([]byte(`{"id":1}`)))
	handlerCtx.QueryProduct(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreatCustomerSuccess(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	creatCustomerSQL := "INSERT INTO \"products\" .+ VALUES .+"
	newRows, _ := util.ObjectToRows(testProduct)
	mock.ExpectBegin()
	mock.ExpectQuery(creatCustomerSQL).WillReturnRows(newRows)
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateProductsRequest{Products: &[]model.Product{testProduct}})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateProducts(w, r)

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
	reqBody, _ := json.Marshal(CreateProductsRequest{Products: &[]model.Product{testProduct}})
	r := httptest.NewRequest(http.MethodGet, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateProducts(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCreatCustomerMissingName(t *testing.T) {
	sqlDB, _, _ := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	customerWithoutName := testProduct
	customerWithoutName.Name = ""
	products := []model.Product{
		customerWithoutName,
	}
	requestObj := CreateProductsRequest{
		Products: &products,
	}

	reqBody, _ := json.Marshal(requestObj)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateProducts(w, r)

	acutalResp := model.Customer{}
	json.Unmarshal(w.Body.Bytes(), &acutalResp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatCustomerNameExisting(t *testing.T) {
	sqlDB, _, mock := util.DbMock(t)
	defer sqlDB.Close()
	handlerCtx := HandlerContext{}
	handlerCtx.InitialHandlerContext(dal.Q)

	creatSQL := "INSERT INTO \"products\" .+ VALUES .+"
	mock.ExpectBegin()
	mock.ExpectQuery(creatSQL).WillReturnError(gorm.ErrDuplicatedKey)

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(CreateProductsRequest{Products: &[]model.Product{testProduct}})
	r := httptest.NewRequest(http.MethodPost, "http://localhosts", bytes.NewBuffer(reqBody))
	handlerCtx.CreateProducts(w, r)

	assert.Nil(t, mock.ExpectationsWereMet())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
