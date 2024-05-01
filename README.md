# Order_system
An simple Order system based on Golang.

## Before start
> - Make sure Golang(1.21+) is installed.
> - Make sure docker is installed and running on your machine.
> - By default app will expose port **8088** and **8089**, and postgres port is **5433**.

## Download Repository
Download this repo to you local machine, open your terminal and navigate to **order_system** directory.
> All following commands must be executed in **order_system** directory.

## Build image
Run following commands to build Order and Payment images:
```
docker build . -t my-order-app -f ./docker/order/Dockerfile
docker build . -t my-payment-app -f ./docker/payment/Dockerfile
```

For Windows:
```
docker build . -t my-order-app -f .\docker\order\Dockerfile
docker build . -t my-payment-app -f .\docker\payment\Dockerfile
```

## Start API services
Run following commands to start the APIs, it will start 3 containers, including Postgres,Order App and Payment App.
```
docker compose up -d
```

## Unit test
To run unit test cases, Please run following command
```
go test ./...
```

## Payload for API testing
- create_customer
```
curl --location 'http://0.0.0.0:8088/order/create_customer' \
--header 'Content-Type: application/json' \
--data-raw '{
    "customers": [
        {
            "name": "Test Customer",
            "email": "test@gmail.com",
            "address": "2 Chartwell Ln"
        }
    ]
}'
```
- query_customer
```
curl --location --request GET 'http://0.0.0.0:8088/order/query_customer' \
--header 'Content-Type;' \
--data '{
    "customer_id": 1
}'
```
- create_product
```
curl --location 'http://0.0.0.0:8088/order/create_product' \
--header 'Content-Type: application/json' \
--data '{
    "products": [
        {
            "name": "Product",
            "description": "this is demo product",
            "price": 10.00,
            "is_available": true
        }
    ]
}'
```
- query_product
```
curl --location --request GET 'http://0.0.0.0:8088/order/query_product' \
--header 'Content-Type: application/json' \
--data '{
    "id":1
}'
```
- create_order
```
curl --location 'http://0.0.0.0:8088/order/create_order' \
--header 'Content-Type: application/json' \
--data '{
    "customer_id":1,
    "product_id":1
}'
```
- query_order
```
curl --location --request GET 'http://0.0.0.0:8088/order/query_order' \
--header 'Content-Type: application/json' \
--data '{
    "id":1
}'
```
- payment_callback
```
curl --location 'http://0.0.0.0:8088/order/payment_callback' \
--header 'Content-Type: application/json' \
--data '{
    "order_id":1,
    "payment_detail":{
        "id":1,
        "amount": 10.00,
        "state": 1
    }
}'
```
- new_payment
```
curl --location 'http://0.0.0.0:8089/payment/new_payment' \
--header 'Content-Type: application/json' \
--data '{
    "id": 1,
    "customer_id": 1,
    "product_id": 1,
    "amount": 10.00,
    "state": 1
}'
```