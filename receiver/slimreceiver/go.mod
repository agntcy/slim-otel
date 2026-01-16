module github.com/agntcy/slim/otel/receiver/slimreceiver

go 1.25.5

replace github.com/agntcy/slim/otel => ../../

replace github.com/agntcy/slim/otel/internal/sharedcomponent => ../../internal/sharedcomponent

replace github.com/agntcy/slim/bindings/generated => /Users/micpapal/Documents/code/agntcy/slim/data-plane/bindings/go/generated

require (
	github.com/agntcy/slim/bindings/generated v0.0.0-00010101000000-000000000000
	github.com/agntcy/slim/otel v0.0.0
	github.com/agntcy/slim/otel/internal/sharedcomponent v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector/component v1.49.0
	go.opentelemetry.io/collector/consumer v1.48.0
	go.opentelemetry.io/collector/pdata v1.49.0
	go.opentelemetry.io/collector/receiver v1.48.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	go.opentelemetry.io/collector/featuregate v1.49.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.49.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)
