package main

import (
	"fmt"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"order_system/custom/message_queue"
	"order_system/custom/payment"
	"order_system/custom/util"
	"order_system/dal"
	"order_system/model"
	"time"
)

func main() {
	serverConfig := util.ServerConfig{}
	serverConfig.GetConf("./config/config.yaml")
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		serverConfig.Postgres.Host, serverConfig.Postgres.Port, serverConfig.Postgres.Username, serverConfig.Postgres.Password, serverConfig.Postgres.Database)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database" + err.Error())
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	dal.SetDefault(db)

	// Auto migrate table schemas
	err = db.AutoMigrate(model.Payment{})
	if err != nil {
		panic("failed to migrate database" + err.Error())
	}

	// Initialize handler context
	paymentCtx := payment.HandlerContext{}
	paymentCtx.InitialHandlerContext(dal.Q,
		message_queue.NewMessageQueue(),
		paymentCtx.ProcessPaymentMethod,
		serverConfig.Order_payment_callback_url,
		paymentCtx.CallPaymentCallbackAPI)

	go paymentCtx.ConsumePaymentMQ()

	http.HandleFunc("/payment/new_payment", paymentCtx.PublishPaymentMQ)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", serverConfig.Payment_port), nil))
}
