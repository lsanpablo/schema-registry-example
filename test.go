package types

import "encoding/json"

type Items struct {
	ProductId  string      `json:"product_id" zog:"productid"`
	Quantity   int64       `json:"quantity" zog:"quantity"`
	TotalPrice json.Number `json:"total_price" zog:"totalprice"`
	UnitPrice  json.Number `json:"unit_price" zog:"unitprice"`
}

type ShippingAddress struct {
	City       string `json:"city" zog:"city"`
	Country    string `json:"country" zog:"country"`
	PostalCode string `json:"postal_code" zog:"postalcode"`
	State      string `json:"state" zog:"state"`
	Street     string `json:"street" zog:"street"`
}

type Data struct {
	CreatedAt       string          `json:"created_at" zog:"createdat"`
	CustomerId      string          `json:"customer_id" zog:"customerid"`
	Items           []Items         `json:"items" zog:"items"`
	OrderId         string          `json:"order_id" zog:"orderid"`
	OrderStatus     string          `json:"order_status" zog:"orderstatus"`
	ShippingAddress ShippingAddress `json:"shipping_address" zog:"shippingaddress"`
	ShippingMethod  string          `json:"shipping_method,omitempty" zog:"shippingmethod"`
	TotalAmount     json.Number     `json:"total_amount" zog:"totalamount"`
}

type Metadata struct {
	EventId       string `json:"event_id" zog:"eventid"`
	EventType     string `json:"event_type" zog:"eventtype"`
	SchemaVersion string `json:"schema_version" zog:"schemaversion"`
	Timestamp     string `json:"timestamp" zog:"timestamp"`
	Version       int64  `json:"version" zog:"version"`
}

type Root struct {
	Data     Data     `json:"data" zog:"data"`
	Metadata Metadata `json:"metadata" zog:"metadata"`
}
