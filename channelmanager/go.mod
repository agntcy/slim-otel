module github.com/agntcy/slim-otel/channelmanager

go 1.26.1

replace github.com/agntcy/slim-otel => ../

replace github.com/agntcy/slim-otel/slimconfig => ../slimconfig

replace github.com/agntcy/slim-otel/internal/sharedcomponent => ../internal/sharedcomponent

require (
	github.com/agntcy/slim-bindings-go v1.2.0
	github.com/agntcy/slim-otel v0.3.1
	github.com/agntcy/slim-otel/slimconfig v0.3.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
)
