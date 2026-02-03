# Channel Manager CTL

A simple command-line client for interacting with the Channel Manager gRPC service.

## Building

```bash
task channelmanager:cmctl:build
```

Or using go directly:
```bash
cd channelmanager/cmd/cmctl
go build -o cmctl
```

## Usage

```bash
./cmctl <command> [channel] [participant] [options]
```

### Positional Arguments

- `command`: Command to execute (required)
- `channel`: Channel name (required for channel-specific commands)
- `participant`: Participant name (required for participant commands)

### Options

- `-server`: gRPC server address (default: `localhost:46358`)
- `-disable-mls`: Disable MLS for channel creation (MLS is enabled by default)

### Available Commands

#### Create a new channel
```bash
./cmctl create-channel org/ns/channel
```

Create a channel with MLS disabled:
```bash
./cmctl create-channel org/ns/channel -disable-mls
```

#### Delete a channel
```bash
./cmctl delete-channel org/ns/channel
```

#### Add a participant to a channel
```bash
./cmctl add-participant org/ns/channel agntcy/ns/participant
```

#### Remove a participant from a channel
```bash
./cmctl delete-participant org/ns/channel agntcy/ns/participant
```

#### List all channels (returns only the list handled by this channel-manager)
```bash
./cmctl list-channels
```

#### List participants in a channel
```bash
./cmctl list-participants org/ns/channel
```

### Examples

Connect to a different server:
```bash
./cmctl list-channels -server "192.168.1.100:46358"
```

Create a channel and add participants:
```bash
# Create channel with MLS enabled (default)
./cmctl create-channel team-chat

# Add participants
./cmctl add-participant org/ns/channel org/ns/participant-1
./cmctl add-participant org/ns/channel org/ns/participant-2

# List participants
./cmctl list-participants org/ns/channel
```
