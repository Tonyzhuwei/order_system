package main

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"order_system/custom/customer"
	"order_system/custom/order"
	"order_system/custom/product"
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

	// Auto migrate table schemas
	err = db.AutoMigrate(model.ALL_ORDER_TABLES...)
	if err != nil {
		panic("failed to migrate database" + err.Error())
	}

	// Initialize handler contexts
	dal.SetDefault(db)
	customerCtx := customer.HandlerContext{}
	customerCtx.InitialHandlerContext(dal.Q)
	productCtx := product.HandlerContext{}
	productCtx.InitialHandlerContext(dal.Q)
	orderCtx := order.HandlerContext{}
	orderCtx.InitialHandlerContext(dal.Q, orderCtx.CallPaymentApi, serverConfig.Payment_message_queue_url)

	// Execute orders
	go orderCtx.ScanPendingOrders()
	go orderCtx.ExecuteOrders()

	// Start REST APIs

	http.HandleFunc("/order/create_customer", customerCtx.CreateCustomers)
	http.HandleFunc("/order/query_customer", customerCtx.QueryCustomer)
	http.HandleFunc("/order/create_product", productCtx.CreateProducts)
	http.HandleFunc("/order/query_product", productCtx.QueryProduct)
	http.HandleFunc("/order/create_order", orderCtx.CreateOrder)
	http.HandleFunc("/order/query_order", orderCtx.QueryOrder)
	http.HandleFunc("/order/payment_callback", orderCtx.PaymentCallBack)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", serverConfig.Order_port), nil))
}
