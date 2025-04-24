generate-go-proto:
	cd service && protoc --go_out=. --go-grpc_out=. proto/schema_registry.proto