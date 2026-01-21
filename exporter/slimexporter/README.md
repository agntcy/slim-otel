# SLIM Exporter

The SLIM exporter sends OpenTelemetry traces, metrics, and logs to a [SLIM](https://github.com/agntcy/slim) (Secure Low-Latency Interactive Messaging) channel. SLIM facilitates secure, low-latency communication between ai-agents and applications using various communication patterns such as point-to-point or groups. This exporter enables secure distribution of observability data using SLIM's end-to-end encryption with MLS (Message Layer Security) and flexible channel-based routing.

## Configuration settings

The following settings are required:

- `endpoint` (default = `http://127.0.0.1:46357`): The address of the SLIM node to connect to.
- `shared-secret` (no default): The shared secret used for MLS and identity provider authentication.

The following settings can be optionally configured:

- `exporter-names`: Names for each signal type exporter. Each exporter name identifies this collector instance in SLIM channels.
  - `metrics` (default = `agntcy/otel/exporter-metrics`): Name for the metrics exporter.
  - `traces` (default = `agntcy/otel/exporter-traces`): Name for the traces exporter.
  - `logs` (default = `agntcy/otel/exporter-logs`): Name for the logs exporter.
- `channels` (default = `[]`): A list of channel configurations to create. When the list is empty, the exporter operates in passive mode, only listening for invitations from other participants. When channels are configured, the exporter actively creates those channels and invites participants, while also continuing to listen for incoming invitations from other participants.

### Channel Configuration

Each channel in the `channels` array supports the following configuration:

- `channel-name` (required): The name of the SLIM channel in the form `org/namespace/service`.
- `signal` (required): The signal type for this channel. Valid values are `traces`, `metrics`, or `logs`. Each channel handles one signal type.
- `participants` (required): An array of participant identifiers to invite to the channel.
- `mls-enabled` (default = `false`): Flag to enable or disable MLS (Message Layer Security) encryption for this channel.

### Example configuration

Example configuration with multiple channels:

```yaml
exporters:
  slim:
    endpoint: "http://127.0.0.1:46357"
    exporter-names:
      metrics: "agntcy/otel/exporter-metrics"
      traces: "agntcy/otel/exporter-traces"
      logs: "agntcy/otel/exporter-logs"
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"
    channels: 
      - channel-name: "agntcy/otel/channel-metrics"
        signal: metrics
        participants:
          - "agntcy/otel/receiver-app"
        mls-enabled: false
      
      - channel-name: "agntcy/otel/channel-traces"
        signal: traces
        participants:
          - "agntcy/otel/receiver-app"
        mls-enabled: true
      
      - channel-name: "agntcy/otel/channel-logs"
        signal: logs
        participants:
          - "agntcy/otel/receiver-app"
        mls-enabled: false
```

Example configuration listening for invitations:

```yaml
exporters:
  slim:
    endpoint: "http://127.0.0.1:46357"
    exporter-names:
      metrics: "agntcy/otel/passive-exporter-metrics"
      traces: "agntcy/otel/passive-exporter-traces"
      logs: "agntcy/otel/passive-exporter-logs"
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"
    # No channels defined - will listen for invitations
```

Complete pipeline configuration:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  slim:
    endpoint: "http://127.0.0.1:46357"
    exporter-names:
      metrics: "agntcy/otel/exporter-metrics"
      traces: "agntcy/otel/exporter-traces"
      logs: "agntcy/otel/exporter-logs"
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"
    channels: 
      - channel-name: "agntcy/otel/channel-traces"
        signal: traces
        participants:
          - "agntcy/otel/receiver-app"
        mls-enabled: true
      
      - channel-name: "agntcy/otel/channel-metrics"
        signal: metrics
        participants:
          - "agntcy/otel/receiver-app"
        mls-enabled: true
      
      - channel-name: "agntcy/otel/channel-logs"
        signal: logs
        participants:
          - "agntcy/otel/receiver-app"
        mls-enabled: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [slim]
    
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [slim]
    
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [slim]
```

## How it works

The SLIM exporter:

1. **Connects** to a SLIM node using the configured endpoint and authenticates using the shared secret.
2. **Creates channels** based on the `channels` configuration. Each channel is created with its specified name and handles one signal type.
3. **Invites participants** to the created channels if configured.
4. **Publishes** OpenTelemetry data (serialized as protobuf) to the appropriate SLIM channels based on signal type.
5. **Listens for invitations** from other participants to join additional channels.

The exporter can operate in multiple modes simultaneously: it can create and manage its own channels while also accepting invitations to join channels created by other participants.

### Signal Routing

Each signal type (traces, metrics, logs) is routed to channels configured for that specific signal type. You must create separate channels for each signal type you want to export:
- Traces → channels with `signal: traces`
- Metrics → channels with `signal: metrics`
- Logs → channels with `signal: logs`

This allows for fine-grained control over which participants receive which types of telemetry data.

### Security

The SLIM exporter supports end-to-end encryption through MLS (Message Layer Security - RFC 9420) when `mls-enabled` is set to `true` for a channel.

## Additional Information

- [SLIM Project](https://github.com/agntcy/slim)
- [SLIM Documentation](https://docs.agntcy.org/messaging/slim-core/)
- [MLS RFC 9420](https://datatracker.ietf.org/doc/rfc9420/)
