module github.com/agntcy/slim/otel/channelmanager

go 1.25.5

replace github.com/agntcy/slim/otel => ../

replace github.com/agntcy/slim/otel/internal/sharedcomponent => ../internal/sharedcomponent

require (
	github.com/agntcy/slim/otel v0.0.0-00010101000000-000000000000
	github.com/mitchellh/mapstructure v1.5.0
	go.uber.org/zap v1.27.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/agntcy/slim-bindings-go v0.7.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
