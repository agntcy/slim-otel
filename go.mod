module github.com/agntcy/slim/otel

go 1.25.5

replace github.com/agntcy/slim/bindings/generated => /Users/micpapal/Documents/code/agntcy/slim/data-plane/bindings/go/generated

require (
	github.com/agntcy/slim/bindings/generated v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.1
)

require (
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)
