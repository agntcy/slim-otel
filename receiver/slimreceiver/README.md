# SLIM Receiver

The SLIM receiver receives OpenTelemetry traces, metrics, and logs from [SLIM](https://github.com/agntcy/slim) (Secure Low-Latency Interactive Messaging) channels. SLIM facilitates secure, low-latency communication between ai-agents and applications using various communication patterns such as point-to-point or groups. This receiver enables secure collection of observability data using SLIM's end-to-end encryption with MLS (Message Layer Security) and flexible session-based routing.

## Configuration settings

The following settings are required:

- `endpoint` (default = `http://127.0.0.1:46357`): The address of the SLIM node to connect to.
- `shared-secret` (no default): The shared secret used for MLS and identity provider authentication.

The following settings can be optionally configured:

- `receiver-name` (default = `agntcy/otel/receiver`): Name for the receiver to be used in SLIM channels. This is the identifier that other participants use to establish sessions with this receiver.

## Example configuration

Example receiver configuration:

```yaml
receivers:
  slim:
    endpoint: "http://127.0.0.1:46357"
    receiver-name: "agntcy/otel/receiver"
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  debug:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200

service:
  pipelines:
    traces:
      receivers: [slim]
      processors: [batch]
      exporters: [debug]
    
    metrics:
      receivers: [slim]
      processors: [batch]
      exporters: [debug]
    
    logs:
      receivers: [slim]
      processors: [batch]
      exporters: [debug]
```

Example with custom receiver name:

```yaml
receivers:
  slim:
    endpoint: "http://127.0.0.1:46357"
    receiver-name: "my-org/my-app/telemetry-receiver"
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"
```

Complete pipeline configuration with multiple exporters:

```yaml
receivers:
  slim:
    endpoint: "http://127.0.0.1:46357"
    receiver-name: "agntcy/otel/receiver"
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  debug:
    verbosity: detailed
  
  otlp:
    endpoint: "backend:4317"
    tls:
      insecure: true
  
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    traces:
      receivers: [slim]
      processors: [batch]
      exporters: [debug, otlp]
    
    metrics:
      receivers: [slim]
      processors: [batch]
      exporters: [debug, otlp, prometheus]
    
    logs:
      receivers: [slim]
      processors: [batch]
      exporters: [debug, otlp]
```

## How it works

The SLIM receiver:

1. **Connects** to a SLIM node using the configured endpoint and authenticates using the shared secret.
2. **Registers** as an application with the configured `receiver-name`, making it discoverable to other SLIM participants.
3. **Listens** for incoming SLIM sessions from any participant that wants to send telemetry data.
4. **Detects signal type** automatically by attempting to unmarshal received data as traces, metrics, or logs.
5. **Routes** the telemetry data to the appropriate consumer (traces, metrics, or logs) based on the detected signal type.
6. **Supports multiple concurrent sessions** from different senders simultaneously.

The receiver operates in a passive listening mode, accepting sessions from any authenticated participant. This allows multiple exporters or applications to send telemetry data to a single receiver instance.

### Session Management

Each incoming SLIM session is handled independently:
- Sessions are processed concurrently in separate goroutines
- Each session can send multiple messages
- Sessions remain open until the sender closes them or an error occurs
- The receiver tracks all active sessions and gracefully closes them during shutdown

### Security

The SLIM receiver supports end-to-end encryption through MLS (Message Layer Security - RFC 9420). When a sender initiates an MLS-encrypted session, the receiver automatically participates in the MLS protocol using the configured shared secret for authentication.

All session establishment and message exchange use SLIM's security features, including:
- Identity verification using the shared secret
- Optional MLS encryption for end-to-end security
- Secure session lifecycle management

## Additional Information

- [SLIM Project](https://github.com/agntcy/slim)
- [SLIM Documentation](https://docs.agntcy.org/messaging/slim-core/)
- [MLS RFC 9420](https://datatracker.ietf.org/doc/rfc9420/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)

