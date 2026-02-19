module github.com/agntcy/slim/otel/sdkexporters/slimtrace/example

go 1.25.7

replace github.com/agntcy/slim/otel/sdkexporters/slimtrace => ../

replace github.com/agntcy/slim/otel => ../../../

require (
	github.com/agntcy/slim/otel v0.0.0
	github.com/agntcy/slim/otel/sdkexporters/slimtrace v0.0.0
	go.opentelemetry.io/otel v1.33.0
	go.opentelemetry.io/otel/sdk v1.33.0
)

require (
	github.com/agntcy/slim-bindings-go v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.33.0 // indirect
	go.opentelemetry.io/otel/trace v1.33.0 // indirect
	go.opentelemetry.io/proto/otlp v1.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/grpc v1.68.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
