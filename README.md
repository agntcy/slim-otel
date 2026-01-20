# SLIM OpenTelemetry Collector

A custom distribution of the OpenTelemetry Collector with the SLIM exporter for sending observability data over secure, low-latency SLIM channels.

## Building the Collector

To build the SLIM OpenTelemetry Collector:

```bash
task collector:build
```

This command will:
1. Download OCB if not already present
2. Generate collector sources based on `builder-config.yaml`
3. Output the binary to `./slim-otelcol/slim-otelcol`

## Running the Collector

### Run Locally

To run the SLIM OpenTelemetry Collector with the default configuration:

```bash
task collector:run
```

The collector will use the configuration defined in `example-collector-config.yaml`.

## Testing

### Run Exporter Tests

To run unit tests for the SLIM exporter:

```bash
task collector:test
```

### End-to-End Testing

#### 1. Start a SLIM Node

First, start a SLIM node for testing in Docker:

```bash
task test:slim:run
```

This runs a SLIM data plane node in a Docker container using the configuration from `slim-test-config.yaml`. The node will be accessible on port 46357.

#### 2. Start the Collector

In a separate terminal, start the SLIM OpenTelemetry Collector with the `example-collector-config.yaml` configuration:

```bash
task collector:run
```

#### 3. Run Test Applications

The test application can be run in two modes:

**Participant Mode**: The collector invites the participant to the channels specified in `example-collector-config.yaml`:

```bash
task test:app:participant
```

The collector must have the participant configured in its channels list in `example-collector-config.yaml`.

**Initiator Mode**: The test app creates channels and invites the collector to join:

```bash
# Get all signals (traces, metrics, logs)
task test:app:get-all-signals

# Or get specific signals:
task test:app:get-metrics
task test:app:get-traces
```

When using initiator mode, the collector's channels list in `example-collector-config.yaml` can be set to `[]` since the app will invite the collector dynamically.

In both cases, telemetry data will flow through the SLIM collector over secure SLIM channels and will be logged by the test application.

## Configuration

The SLIM collector can be configured using the `example-collector-config.yaml` file. Key configuration options include:

- `endpoint`: Address of the SLIM node (default: `http://127.0.0.1:46357`)
- `exporter-names`: Separate names for metrics, traces, and logs exporters
- `shared-secret`: Shared secret for MLS and identity provider authentication
- `channels`: Array of channel configurations (can be empty for passive/listen-only mode)

See `example-collector-config.yaml` for a complete example and the README.md in `./exporter/slimexporter` for detailed configuration options.
