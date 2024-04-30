package util

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/romana/rlog"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"log"
	"net/http"
	"order_system/dal"
	"os"
	"testing"
)

type DbConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type ServerConfig struct {
	Postgres                   DbConfig `yaml:"postgres"`
	Order_payment_callback_url string   `yaml:"order_payment_callback_url"`
	Payment_message_queue_url  string   `yaml:"payment_message_queue_url"`
}

func (c *ServerConfig) GetConf(fileName string) *ServerConfig {
	yamlFile, err := os.ReadFile(fileName)
	if err != nil {
		log.Printf("Read yaml file %s failed: %s ", fileName, err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func IsAllowHttpMethod(methods []string, w http.ResponseWriter, r *http.Request) bool {
	for _, method := range methods {
		if method == r.Method {
			return true
		}
	}
	http.Error(w, "Not allow http method", http.StatusMethodNotAllowed)
	return false
}

func FetchReqObject(r *http.Request, reqObj interface{}) error {
	if r == nil {
		return errors.New("http request is nil")
	}
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		errInfo := "Read request body failed" + err.Error()
		rlog.Error(errInfo)
		return errors.New(errInfo)
	}
	err = json.Unmarshal(reqBody, reqObj)
	if err != nil {
		errInfo := "Unmarshal request body failed" + err.Error()
		rlog.Error(errInfo)
		return errors.New(errInfo)
	}
	return nil
}

func GetStringPtr(s string) *string {
	return &s
}

// DbMock For unit test usage
func DbMock(t *testing.T) (*sql.DB, *gorm.DB, sqlmock.Sqlmock) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	gormdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqldb,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		t.Fatal(err)
	}

	dal.SetDefault(gormdb)

	return sqldb, gormdb, mock
}

// ObjectToRows For unit test usage
func ObjectToRows(object interface{}) (*sqlmock.Rows, error) {
	buf, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	rowMap := make(map[string]interface{})
	err = json.Unmarshal(buf, &rowMap)
	if err != nil {
		return nil, err
	}
	columns := make([]string, 0)
	values := make([]driver.Value, 0)
	for k, v := range rowMap {
		columns = append(columns, k)
		values = append(values, v)
	}
	return sqlmock.NewRows(columns).AddRow(values...), nil
}
