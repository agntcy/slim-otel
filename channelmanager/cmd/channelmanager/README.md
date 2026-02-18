# Channel Manager

A go application for managing SLIM channels and participants. The Channel Manager creates and maintains SLIM sessions (channels) and handles participant invitations based on a configuration file.

## Overview

The Channel Manager:
- Connects to a SLIM node and creates channels defined in the configuration
- Invites participants to channels automatically on startup
- Exposes a gRPC API for dynamic channel and participant management

## Building

```bash
task channelmanager:build
```

## Configuration

Create a YAML configuration file (see [example-channel-manager-config.yaml](example-channel-manager-config.yaml)):

```yaml
channel-manager:
  # Connection configuration for SLIM node
  connection-config:
    endpoint: "http://127.0.0.1:46357"
    
  # gRPC service address for accepting commands
  service-address: "127.0.0.1:46358"
  
  # Name of the channel manager in SLIM
  local-name: "agntcy/otel/channel-manager"
  
  # Shared secret for MLS and identity provider
  shared-secret: "your-shared-secret-here"

# Channels to create on startup
channels:
  - name: "agntcy/otel/channel"
    participants:
      - "agntcy/otel/exporter-traces"
      - "agntcy/otel/receiver"
    mls-enabled: true
```

## Running

Start the channel manager with a configuration file:

```bash
./channelmanager -config-file config.yaml
```