package constants

// Payment State
const PAYMENT_STATE_INPROGRESS = int8(0)
const PAYMENT_STATE_SUCCESS = int8(1)
const PAYMENT_STATE_FAILED = int8(2)
const PAYMENT_STATE_REFUND = int8(3)

// Error responses
const PRODUCT_NOT_AVAILABLE = "product not available"
const CREATE_ORDER_FAILED = "create order failed"
