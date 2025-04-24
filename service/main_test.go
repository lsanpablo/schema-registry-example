package main

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"net"
	schemaregistry "service/schemaregistrygrpc"
	"testing"
)

const bufSize = 1024 * 1024

type TestOrderCreated struct {
	Data struct {
		CreatedAt  string `json:"created_at"`
		CustomerId string `json:"customer_id"`
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
		TotalAmount    json.Number `json:"total_amount"`
	} `json:"data"`
	Metadata struct {
		EventId       string `json:"event_id"`
		EventType     string `json:"event_type"`
		SchemaVersion string `json:"schema_version"`
		Timestamp     string `json:"timestamp"`
		Version       int64  `json:"version"`
	} `json:"metadata"`
}

func newTestServer(t *testing.T) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(bufSize)

	s := grpc.NewServer()
	schemaMap := GetSchemaMap()

	svc := &server{
		schemaMap: schemaMap,
	}

	schemaregistry.RegisterSchemaRegistryServer(s, svc)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("gRPC server exited: %v", err)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()), // or WithInsecure()
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	cleanup := func() {
		conn.Close()
		s.Stop()
	}

	return conn, cleanup
}

func TestValidateEvent_Success(t *testing.T) {
	conn, cleanup := newTestServer(t)
	defer cleanup()

	client := schemaregistry.NewSchemaRegistryClient(conn)

	event := TestOrderCreated{
		Data: struct {
			CreatedAt  string `json:"created_at"`
			CustomerId string `json:"customer_id"`
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
			TotalAmount    json.Number `json:"total_amount"`
		}{
			CreatedAt:  "2025-04-23T15:04:05Z",
			CustomerId: "cust_abc123",
			Items: []struct {
				ProductId  string      `json:"product_id"`
				Quantity   int64       `json:"quantity"`
				TotalPrice json.Number `json:"total_price"`
				UnitPrice  json.Number `json:"unit_price"`
			}{
				{
					ProductId:  "prod_001",
					Quantity:   2,
					TotalPrice: json.Number("19.98"),
					UnitPrice:  json.Number("9.99"),
				},
				{
					ProductId:  "prod_002",
					Quantity:   1,
					TotalPrice: json.Number("29.99"),
					UnitPrice:  json.Number("29.99"),
				},
			},
			OrderId:     "order_456xyz",
			OrderStatus: "processing",
			ShippingAddress: struct {
				City       string `json:"city"`
				Country    string `json:"country"`
				PostalCode string `json:"postal_code"`
				State      string `json:"state"`
				Street     string `json:"street"`
			}{
				City:       "San Francisco",
				Country:    "US",
				PostalCode: "94107",
				State:      "CA",
				Street:     "123 Market St",
			},
			ShippingMethod: "standard",
			TotalAmount:    json.Number("49.97"),
		},
		Metadata: struct {
			EventId       string `json:"event_id"`
			EventType     string `json:"event_type"`
			SchemaVersion string `json:"schema_version"`
			Timestamp     string `json:"timestamp"`
			Version       int64  `json:"version"`
		}{
			EventId:       "evt_789xyz",
			EventType:     "order.created",
			SchemaVersion: "1.0.0",
			Timestamp:     "2025-04-23T15:04:05Z",
			Version:       1,
		},
	}

	payload, _ := json.Marshal(event)

	resp, err := client.ValidateEvent(context.Background(), &schemaregistry.ValidateEventRequest{
		EventSchemaId: "order.created",
		Payload:       payload,
		Format:        schemaregistry.Format_FORMAT_JSON,
	})

	if err != nil {
		t.Fatalf("ValidateEvent failed: %v", err)
	}

	if !resp.Valid {
		t.Errorf("Expected valid=true, got false: %s", resp.Message)
	}
}

func TestValidateEvent_Failures(t *testing.T) {
	conn, cleanup := newTestServer(t)
	defer cleanup()

	client := schemaregistry.NewSchemaRegistryClient(conn)

	testCases := []struct {
		name        string
		event       TestOrderCreated
		eventSchema string
		wantValid   bool
	}{
		{
			name: "invalid customer id",
			event: TestOrderCreated{
				Data: struct {
					CreatedAt  string `json:"created_at"`
					CustomerId string `json:"customer_id"`
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
					TotalAmount    json.Number `json:"total_amount"`
				}{
					CreatedAt:  "2025-04-23T15:04:05Z",
					CustomerId: "bad id",
					Items: []struct {
						ProductId  string      `json:"product_id"`
						Quantity   int64       `json:"quantity"`
						TotalPrice json.Number `json:"total_price"`
						UnitPrice  json.Number `json:"unit_price"`
					}{
						{
							ProductId:  "prod_001",
							Quantity:   2,
							TotalPrice: json.Number("19.98"),
							UnitPrice:  json.Number("9.99"),
						},
						{
							ProductId:  "prod_002",
							Quantity:   1,
							TotalPrice: json.Number("29.99"),
							UnitPrice:  json.Number("29.99"),
						},
					},
					OrderId:     "order_456xyz",
					OrderStatus: "processing",
					ShippingAddress: struct {
						City       string `json:"city"`
						Country    string `json:"country"`
						PostalCode string `json:"postal_code"`
						State      string `json:"state"`
						Street     string `json:"street"`
					}{
						City:       "San Francisco",
						Country:    "US",
						PostalCode: "94107",
						State:      "CA",
						Street:     "123 Market St",
					},
					ShippingMethod: "standard",
					TotalAmount:    json.Number("49.97"),
				},
				Metadata: struct {
					EventId       string `json:"event_id"`
					EventType     string `json:"event_type"`
					SchemaVersion string `json:"schema_version"`
					Timestamp     string `json:"timestamp"`
					Version       int64  `json:"version"`
				}{
					EventId:       "evt_789xyz",
					EventType:     "order.created",
					SchemaVersion: "1.0.0",
					Timestamp:     "2025-04-23T15:04:05Z",
					Version:       1,
				},
			},
			eventSchema: "order.created",
			wantValid:   false,
		},
		{
			name: "bad event id",
			event: TestOrderCreated{
				Data: struct {
					CreatedAt  string `json:"created_at"`
					CustomerId string `json:"customer_id"`
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
					TotalAmount    json.Number `json:"total_amount"`
				}{
					CreatedAt:  "2025-04-23T15:04:05Z",
					CustomerId: "customer_123",
					Items: []struct {
						ProductId  string      `json:"product_id"`
						Quantity   int64       `json:"quantity"`
						TotalPrice json.Number `json:"total_price"`
						UnitPrice  json.Number `json:"unit_price"`
					}{
						{
							ProductId:  "prod_001",
							Quantity:   2,
							TotalPrice: json.Number("19.98"),
							UnitPrice:  json.Number("9.99"),
						},
						{
							ProductId:  "prod_002",
							Quantity:   1,
							TotalPrice: json.Number("29.99"),
							UnitPrice:  json.Number("29.99"),
						},
					},
					OrderId:     "order_456xyz",
					OrderStatus: "processing",
					ShippingAddress: struct {
						City       string `json:"city"`
						Country    string `json:"country"`
						PostalCode string `json:"postal_code"`
						State      string `json:"state"`
						Street     string `json:"street"`
					}{
						City:       "San Francisco",
						Country:    "US",
						PostalCode: "94107",
						State:      "CA",
						Street:     "123 Market St",
					},
					ShippingMethod: "standard",
					TotalAmount:    json.Number("49.97"),
				},
				Metadata: struct {
					EventId       string `json:"event_id"`
					EventType     string `json:"event_type"`
					SchemaVersion string `json:"schema_version"`
					Timestamp     string `json:"timestamp"`
					Version       int64  `json:"version"`
				}{
					EventId:       "bad_event_id",
					EventType:     "order.created",
					SchemaVersion: "1.0.0",
					Timestamp:     "2025-04-23T15:04:05Z",
					Version:       1,
				},
			},
			eventSchema: "order.created",
			wantValid:   false,
		},
		{
			name: "bad event id",
			event: TestOrderCreated{
				Data: struct {
					CreatedAt  string `json:"created_at"`
					CustomerId string `json:"customer_id"`
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
					TotalAmount    json.Number `json:"total_amount"`
				}{
					CreatedAt:  "2025-04-23T15:04:05Z",
					CustomerId: "customer_123",
					Items: []struct {
						ProductId  string      `json:"product_id"`
						Quantity   int64       `json:"quantity"`
						TotalPrice json.Number `json:"total_price"`
						UnitPrice  json.Number `json:"unit_price"`
					}{
						{
							ProductId:  "prod_001",
							Quantity:   2,
							TotalPrice: json.Number("19.98"),
							UnitPrice:  json.Number("9.99"),
						},
						{
							ProductId:  "prod_002",
							Quantity:   1,
							TotalPrice: json.Number("29.99"),
							UnitPrice:  json.Number("29.99"),
						},
					},
					OrderId:     "order_456xyz",
					OrderStatus: "processing",
					ShippingAddress: struct {
						City       string `json:"city"`
						Country    string `json:"country"`
						PostalCode string `json:"postal_code"`
						State      string `json:"state"`
						Street     string `json:"street"`
					}{
						City:       "San Francisco",
						Country:    "US",
						PostalCode: "94107",
						State:      "CA",
						Street:     "123 Market St",
					},
					ShippingMethod: "standard",
					TotalAmount:    json.Number("-1"),
				},
				Metadata: struct {
					EventId       string `json:"event_id"`
					EventType     string `json:"event_type"`
					SchemaVersion string `json:"schema_version"`
					Timestamp     string `json:"timestamp"`
					Version       int64  `json:"version"`
				}{
					EventId:       "evt_123",
					EventType:     "order.created",
					SchemaVersion: "1.0.0",
					Timestamp:     "2025-04-23T15:04:05Z",
					Version:       1,
				},
			},
			eventSchema: "order.created",
			wantValid:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("failed to marshal event: %v", err)
			}

			resp, err := client.ValidateEvent(context.Background(), &schemaregistry.ValidateEventRequest{
				EventSchemaId: tc.eventSchema,
				Payload:       payload,
				Format:        schemaregistry.Format_FORMAT_JSON,
			})
			if err != nil {
				t.Fatalf("ValidateEvent failed: %v", err)
			}

			if resp.Valid != tc.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v. Message: %s", tc.wantValid, resp.Valid, resp.Message)
			}
		})
	}
}
func TestValidateEvent_SchemaNotFound(t *testing.T) {
	conn, cleanup := newTestServer(t)
	defer cleanup()

	client := schemaregistry.NewSchemaRegistryClient(conn)

	resp, err := client.ValidateEvent(context.Background(), &schemaregistry.ValidateEventRequest{
		EventSchemaId: "schema not found",
		Payload:       []byte(`{}`),
		Format:        schemaregistry.Format_FORMAT_JSON,
	})

	if err != nil {
		t.Fatalf("ValidateEvent failed: %v", err)
	}

	if resp.Valid {
		t.Errorf("Expected valid=false for unknown schema")
	}
}
