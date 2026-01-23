module github.com/agntcy/slim/otel/channelmanager

go 1.25.5

replace github.com/agntcy/slim/otel => ../

replace github.com/agntcy/slim/otel/internal/sharedcomponent => ../internal/sharedcomponent

replace github.com/agntcy/slim/bindings/generated => /Users/micpapal/Documents/code/agntcy/slim/data-plane/bindings/go/generated

require (
	github.com/agntcy/slim-bindings-go v0.7.4
	github.com/agntcy/slim/otel v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
