package model

import (
	"time"
)

var ALL_ORDER_TABLES []interface{} = []interface{}{
	Customer{}, Product{}, Order{},
}

type Customer struct {
	ID        uint      `json:"id" gorm:"auto_increment;primary_key"`
	Name      string    `json:"name" gorm:"index;unique;not null"`
	Email     *string   `json:"email,omitempty"`
	Address   *string   `json:"address,omitempty"`
	CreatedAt time.Time `json:"createdTime"`
	UpdatedAt time.Time `json:"updatedTime"`
}

type Product struct {
	ID          uint      `json:"id" gorm:"auto_increment;primary_key"`
	Name        string    `json:"name" gorm:"index;unique;not null"`
	Description *string   `json:"description,omitempty"`
	Price       float64   `json:"price" gorm:"type:decimal(10,2); not null"`
	IsAvailable bool      `json:"is_available" gorm:"not null"`
	CreatedAt   time.Time `json:"createdTime"`
	UpdatedAt   time.Time `json:"updatedTime"`
}

type Order struct {
	ID         uint      `json:"id" gorm:"auto_increment;primary_key"`
	CustomerId uint      `json:"customer_id" gorm:"index;"`
	ProductId  uint      `json:"product_id" gorm:"index;not null"`
	Amount     float64   `json:"amount" gorm:"type:decimal(10,2); not null"`
	State      int8      `json:"state"`
	FailReason *string   `json:"fail_reason,omitempty"`
	CreatedAt  time.Time `json:"createdTime"`
	UpdatedAt  time.Time `json:"updatedTime"`
}

type Payment struct {
	ID              uint      `json:"id" gorm:"auto_increment;primary_key"`
	OrderId         uint      `json:"order_id" gorm:"index;not null"`
	Amount          float64   `json:"amount" gorm:"type:decimal(10,2); not null"`
	State           int8      `json:"state" gorm:"not null"`
	PaymentResult   *string   `json:"payment_result,omitempty"`
	IsNotifiedOrder bool      `json:"is_notified_order" gorm:"not null"`
	CreatedAt       time.Time `json:"createdTime"`
	UpdatedAt       time.Time `json:"updatedTime"`
}
