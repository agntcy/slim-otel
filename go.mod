module github.com/agntcy/slim/otel

go 1.25.6

replace github.com/agntcy/slim/bindings/generated => /Users/micpapal/Documents/code/agntcy/slim/data-plane/bindings/go/generated

require (
	github.com/agntcy/slim/bindings/generated v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.27.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
