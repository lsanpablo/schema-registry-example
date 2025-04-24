package schemas

import (
	"encoding/json"
	z "github.com/Oudwins/zog"
)

type OrderCreated struct {
	Data struct {
		CreatedAt  string `json:"created_at"`
		Customerid string `json:"customer_id"`
		Items      []struct {
			ProductId  string      `json:"product_id"`
			Quantity   int64       `json:"quantity"`
			TotalPrice json.Number `json:"total_price"`
			UnitPrice  json.Number `json:"unit_price"`
		} `json:"items"`
		OrderId         string `json:"order_id"`
		OrderStatus     string `json:"order_status"`
		ShippingAddress struct {
			City       string `json:"city"`
			Country    string `json:"country"`
			PostalCode string `json:"postal_code"`
			State      string `json:"state"`
			Street     string `json:"street"`
		} `json:"shipping_address"`
		ShippingMethod string      `json:"shipping_method,omitempty"`
		Totalamount    json.Number `json:"total_amount"`
	} `json:"data"`
	Metadata struct {
		Eventid       string `json:"event_id"`
		EventType     string `json:"event_type"`
		SchemaVersion string `json:"schema_version"`
		Timestamp     string `json:"timestamp"`
		Version       int64  `json:"version"`
	} `json:"metadata"`
}

func JSONNumberSchema() *z.StringSchema[json.Number] {
	return &z.StringSchema[json.Number]{}
}

var JSONNumberIsPositiveFloat = func(val any, ctx z.Ctx) bool {
	number, ok := val.(json.Number)
	if !ok {
		return false
	}
	float, err := number.Float64()
	if err != nil {
		return false
	}
	return float > 0
}
