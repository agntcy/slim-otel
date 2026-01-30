# SLIM OpenTelemetry Collector

A custom distribution of the OpenTelemetry Collector with the SLIM exporter and receiver for sending and receiving observability data over secure, low-latency SLIM channels.

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

### Run Exporter Locally

To run the SLIM OpenTelemetry Collector with the exporter configuration:

```bash
task collector:run:exporter
```

The collector will use the configuration defined in `exporter/slimexporter/example-exporter-collector-config.yaml`.

### Run Receiver Locally

To run the SLIM OpenTelemetry Collector with the receiver configuration:

```bash
task collector:run:receiver
```

The collector will use the configuration defined in `receiver/slimreceiver/example-receiver-collector-config.yaml`.

## Testing

### Run Tests

To run unit tests for the SLIM exporter, receiver, and internal components:

```bash
task test
```
