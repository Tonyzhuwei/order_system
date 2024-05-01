package constants

// Payment State
const PAYMENT_STATE_CREATED = int8(0)
const PAYMENT_STATE_SUCCESS = int8(1)
const PAYMENT_STATE_FAILED = int8(2)
const PAYMENT_STATE_REFUND = int8(3)

// Error responses
const CUSTOMER_NOT_FOUND = "customer not found"
const PRODUCT_NOT_AVAILABLE = "product not available"
const CREATE_ORDER_FAILED = "create order failed"
const EXCEED_PAYMENT_LIMIT = "exceed payment limit"
