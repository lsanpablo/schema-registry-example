package main

import (
	"context"
	"encoding/json"
	z "github.com/Oudwins/zog"
	"google.golang.org/grpc"
	"log"
	"net"
	"regexp"
	schemaregistry "service/schemaregistrygrpc"
	"service/schemas"
)

type EventSchema struct {
	ValidationSchema  *z.StructSchema
	StructConstructor func() any
}

type server struct {
	schemaregistry.UnimplementedSchemaRegistryServer
	schemaMap map[string]EventSchema
}

func (s *server) ValidateEvent(ctx context.Context, req *schemaregistry.ValidateEventRequest) (*schemaregistry.ValidateEventResponse, error) {
	log.Printf("Received request: event_id=%s format=%s", req.GetEventSchemaId(), req.GetFormat())

	eventSchema, ok := s.schemaMap[req.GetEventSchemaId()]
	if !ok {
		return &schemaregistry.ValidateEventResponse{
			Valid:   false,
			Message: "Schema not found for event_id",
		}, nil
	}

	structPointer := eventSchema.StructConstructor()
	err := json.Unmarshal(req.GetPayload(), structPointer)
	if err != nil {
		return &schemaregistry.ValidateEventResponse{
			Valid:   false,
			Message: "Failed to unmarshal payload",
		}, nil
	}

	errsMap := eventSchema.ValidationSchema.Validate(structPointer)
	if len(errsMap) > 0 {
		return &schemaregistry.ValidateEventResponse{
			Valid:   false,
			Message: "Validation failed",
		}, nil
	}

	return &schemaregistry.ValidateEventResponse{
		Valid:   true,
		Message: "",
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	schemaMap := GetSchemaMap()
	schemaregistry.RegisterSchemaRegistryServer(s, &server{
		schemaMap: schemaMap,
	})

	log.Println("Schema Registry server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func GetSchemaMap() map[string]EventSchema {
	ret := make(map[string]EventSchema)
	eventRegex := regexp.MustCompile("^evt_[a-zA-Z0-9]+$")
	customerRegex := regexp.MustCompile("^cust_[a-zA-Z0-9]+$")
	ret["order.created"] = EventSchema{
		StructConstructor: func() any {
			return &schemas.OrderCreated{}
		},
		ValidationSchema: z.Struct(z.Schema{
			"metadata": z.Struct(z.Schema{
				"eventid": z.String().Match(eventRegex),
			}),
			"data": z.Struct(z.Schema{
				"customerid":  z.String().Match(customerRegex),
				"totalamount": schemas.JSONNumberSchema().TestFunc(schemas.JSONNumberIsPositiveFloat),
			}),
		}),
	}
	return ret
}
