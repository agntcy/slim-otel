# Channel Manager CTL

A simple command-line client for interacting with the Channel Manager gRPC service.

## Building

```bash
task channelmanager:client:build
```

Or using go directly:
```bash
cd channelmanager/cmtl
go build -o cmctl
```

## Usage

```bash
./cmctl [flags]
```

### Flags

- `-server`: gRPC server address (default: `localhost:46358`)
- `-command`: Command to send
- `-channel`: Channel name (required for channel-specific commands)
- `-participant`: Participant name (required for participant commands)
- `-mls`: Enable MLS for channel creation (default: `false`)

### Available Commands

#### Create a new channel
```bash
./cmctl -command create-channel -channel "org/ns/channel"
```

Create a channel with MLS enabled:
```bash
./cmctl -command create-channel -channel "org/ns/channel" -mls
```

#### Delete a channel
```bash
./cmctl -command delete-channel -channel "org/ns/channel"
```

#### Add a participant to a channel
```bash
./cmctl -command add-participant -channel "org/ns/channel" -participant "agntcy/ns/participant"
```

#### Remove a participant from a channel
```bash
./cmctl -command delete-participant -channel "org/ns/channel" -participant "agntcy/ns/participant"
```

#### List all channels (returns only the list handled by this channel-manager)
```bash
./cmctl -command list-channels
```

#### List participants in a channel
```bash
./cmctl -command list-participants -channel "org/ns/channel"
```

### Examples

Connect to a different server:
```bash
./cmctl -server "192.168.1.100:46358" -command list-channels
```

Create a channel and add participants:
```bash
# Create channel
./cmctl -command create-channel -channel "team-chat"

# Add participants
./cmctl -command add-participant -channel "org/ns/channel" -participant "org/ns/channel"
./cmctl -command add-participant -channel "org/ns/channel" -participant "org/ns/channel"

# List participants
./cmctl -command list-participants -channel "org/ns/channel"
```
