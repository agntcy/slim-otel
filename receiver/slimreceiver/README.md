# SLIM Receiver

This is a custom OpenTelemetry Collector receiver that receives telemetry data (traces, metrics, and logs) from SLIM channels.

## Configuration

Example configuration:

```yaml
receivers:
  slim:
    endpoint: "http://127.0.0.1:46357"
    shared-secret: "your-secret-here"
    interval: "1m"
    receiver-names:
      metrics: "agntcy/otel/receiver-metrics"
      traces: "agntcy/otel/receiver-traces"
      logs: "agntcy/otel/receiver-logs"
    channels:
      - "channel-1"
      - "channel-2"
```

## Configuration Options

- `endpoint` (required): The SLIM server endpoint
- `shared-secret` (optional): Shared secret for authentication
- `interval` (required): How often to poll for telemetry data (e.g., "30s", "1m", "5m")
- `receiver-names`: Channel names for each signal type
  - `metrics`: Channel name for metrics
  - `traces`: Channel name for traces
  - `logs`: Channel name for logs
- `channels` (required): List of channel names to subscribe to

## Implementation Status

This is a skeleton implementation with the following interfaces:

### Implemented

- ✅ `Config` struct with validation
- ✅ `receiver.Factory` implementation
- ✅ `receiver.Traces` interface
- ✅ `receiver.Metrics` interface
- ✅ `receiver.Logs` interface
- ✅ Basic polling mechanism
- ✅ Lifecycle management (Start/Shutdown)

### TODO

- [ ] SLIM connection initialization
- [ ] Actual telemetry reception from SLIM channels
- [ ] Message parsing and conversion to OpenTelemetry format
- [ ] Error handling and retry logic
- [ ] Metrics for receiver health and performance
- [ ] Tests

## Usage

To use this receiver in your collector, add it to your `components.go` file:

```go
import (
    slimreceiver "github.com/agntcy/slim/otel/receiver/slimreceiver"
)

func components() (otelcol.Factories, error) {
    // ...
    
    factories.Receivers, err = receiver.MakeFactoryMap(
        otlpreceiver.NewFactory(),
        slimreceiver.NewFactory(), // Add this line
    )
    
    // ...
}
```

## Development

Key files to implement:

1. **config.go**: Receiver configuration and validation
2. **factory.go**: Factory for creating receiver instances
3. **receiver.go**: Main receiver implementation with telemetry collection logic

The receiver follows the OpenTelemetry Collector receiver pattern:
- Implements `component.Component` interface (Start/Shutdown methods)
- Uses consumers to push received telemetry to the next pipeline component
- Supports all three signal types: traces, metrics, and logs
