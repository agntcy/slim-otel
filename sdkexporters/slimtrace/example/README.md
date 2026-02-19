# SLIM Trace SDK Exporter Example

This example demonstrates how to use the SLIM trace SDK exporter to send OpenTelemetry traces directly from your application to SLIM.

## How to test

1. Run slim 
    ```bash
    task slim:run
    ```

2. Run the test app
   ```bash
   cd example
   go mod tidy
   go run main.go
   ```

3. Run the recevier
    ```bash 
    task collector:run:receiver
    ```

4. Create the config file for the channel manager
    ```yaml
    channel-manager:
    # address of the SLIM node where to connect
    connection-config:
        address: "http://127.0.0.1:46357"
    # grpc service to get commands
    service-address: "127.0.0.1:46358"
    # name of the channel manager to be used in SLIM channels
    local-name: "agntcy/otel/channel-manager"
    # shared secret used for MLS and identity provider
    shared-secret: "a-very-long-shared-secret-0123456789-abcdefg"

    # channels to create
    channels:
    - name: "agntcy/otel/channel"
        participants:
        - "sdk/expoter/traces"
        - "agntcy/otel/receiver"
        mls-enabled: true
    ```

5. Run the channel manager
    ```bash
    go run ./cmd/channelmanager/main.go --config-file config.yaml
    ```
