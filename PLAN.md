# IEC 104 CLI Implementation Plan

This document is the agent-facing implementation plan for `iec-104-cli`.

The goal is to build a script-friendly Go command-line client for IEC 60870-5-104, inspired by `opc-ua-cli`, while respecting that IEC 104 is not shaped like OPC UA. OPC UA has endpoints, security policies, namespaces, and node IDs. IEC 104 has TCP sessions, STARTDT, ASDUs, common addresses, information object addresses, causes of transmission, quality flags, interrogation, spontaneous updates, and control commands.

The CLI should feel familiar to users of `opc-ua-cli`, but the protocol model must remain honest.

## Core principles

1. Keep the CLI boring and scriptable.
2. Default to `config.yaml` in the working directory.
3. Support `--config` for custom config files.
4. Let CLI flags override config values.
5. Use subcommands.
6. Support `table`, `text`, `json`, and `jsonl` where useful.
7. Keep command and setpoint operations dry-run by default.
8. Require `--yes` for anything that sends a control command.
9. Wrap the IEC 104 library behind an internal adapter.
10. Do not leak third-party library types into the CLI layer.
11. Build read-only commands before control commands.
12. Prefer clear errors over clever abstractions.

## Recommended protocol backend

Use `github.com/wendy512/iec104` as the first implementation candidate.

Reasons:

- It provides dedicated client and server packages.
- It supports interrogation, counter interrogation, read command, test command, clock synchronisation, and control commands in its examples.
- It supports common monitoring data types and command types.
- It has a recent enough release history to justify trying it first.

Important rule: wrap it behind `internal/iec104`. The public CLI must depend on local interfaces, not on `github.com/wendy512/iec104/client` directly.

If this backend becomes painful, replace only the adapter. Do not smear the dependency across the codebase like protocol peanut butter.

## Target user flow

```bash
iec-104-cli validate-config
iec-104-cli test-connection
iec-104-cli interrogate
iec-104-cli listen --format jsonl
iec-104-cli watch --point active_power --interval 1s
iec-104-cli command single --ioa 1000 --value on --dry-run
iec-104-cli command single --ioa 1000 --value on --yes
```

## Target command set

### Configuration and diagnostics

```bash
iec-104-cli validate-config
iec-104-cli test-connection
iec-104-cli completions bash
iec-104-cli completions zsh
```

### Read-only protocol operations

```bash
iec-104-cli listen
iec-104-cli interrogate
iec-104-cli watch
iec-104-cli read
```

### Control operations

```bash
iec-104-cli command single
iec-104-cli command double
iec-104-cli setpoint normalized
iec-104-cli setpoint scaled
iec-104-cli setpoint float
iec-104-cli clock-sync
```

## Command semantics

### `validate-config`

Validate local configuration only. It must not connect to a server.

It should catch:

- missing host
- invalid port
- invalid timeout
- invalid common address
- invalid originator address
- duplicate point names
- invalid IOA values
- unsupported point types
- unsupported output format

### `test-connection`

Open a TCP connection and perform the IEC 104 session start sequence.

Expected output should include:

```text
Host: 192.168.1.10
Port: 2404
TCP: ok
IEC104 STARTDT: ok
Result: connected
```

This command should not send control commands.

### `listen`

Connect and print incoming ASDUs until interrupted or until `--duration` expires.

Useful flags:

```bash
--duration 30s
--format jsonl
--point active_power
--ioa 1001
--common-address 1
```

### `interrogate`

Send general interrogation and collect responses until timeout or interrogation completion.

Useful flags:

```bash
--common-address 1
--timeout 10s
--point active_power
--ioa 1001
--format json
```

### `watch`

Maintain latest known values and print matching points repeatedly.

This is polling of the local latest-value cache, not repeated IEC 104 interrogation unless explicitly added later.

Useful flags:

```bash
--interval 1s
--point active_power
--ioa 1001
--stale-after 30s
--format jsonl
```

### `read`

Send IEC 104 read command for a specific IOA where supported by the remote server.

This is different from OPC UA read. Many IEC 104 devices will rely on interrogation and spontaneous updates instead.

### `command single`

Send a single command.

Example:

```bash
iec-104-cli command single --ioa 1000 --value on --yes
```

Allowed values:

```text
on
off
true
false
1
0
```

### `command double`

Send a double command.

Example:

```bash
iec-104-cli command double --ioa 1001 --value close --yes
```

Allowed values:

```text
on
off
open
close
intermediate
indeterminate
```

### `setpoint`

Send setpoint commands.

Supported forms:

```bash
iec-104-cli setpoint normalized --ioa 2000 --value 0.5 --yes
iec-104-cli setpoint scaled --ioa 2001 --value 42 --yes
iec-104-cli setpoint float --ioa 2002 --value 12.5 --yes
```

### `clock-sync`

Send clock synchronisation command.

Require `--yes`. Time changes can upset historical data and event ordering, so this must not be casual finger-drumming.

## Initial config model

```yaml
connection:
  host: 127.0.0.1
  port: 2404
  timeout: 10s
  reconnect: true
  reconnect_interval: 5s

iec104:
  common_address: 1
  originator_address: 0
  interrogation_qualifier: 20

output:
  format: table
  timestamps: local

points:
  - name: active_power
    ioa: 1001
    type: float
    unit: MW

  - name: breaker_closed
    ioa: 2001
    type: single_point
```

## Planned repository structure

```text
.
├── main.go
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── PLAN.md
├── TASKS.md
├── config.example.yaml
├── internal
│   ├── cli
│   │   ├── root.go
│   │   ├── validate_config.go
│   │   ├── test_connection.go
│   │   ├── listen.go
│   │   ├── interrogate.go
│   │   ├── watch.go
│   │   ├── read.go
│   │   ├── command.go
│   │   ├── setpoint.go
│   │   ├── clock_sync.go
│   │   ├── completions.go
│   │   └── output.go
│   ├── config
│   │   ├── config.go
│   │   └── config_test.go
│   ├── iec104
│   │   ├── client.go
│   │   ├── adapter_wendy.go
│   │   ├── types.go
│   │   ├── decode.go
│   │   ├── command.go
│   │   └── errors.go
│   └── exitcode
│       └── exitcode.go
└── .github
    └── workflows
        └── ci.yml
```

## Internal IEC 104 interface

Start with this shape and adjust only when the backend proves otherwise.

```go
type Client interface {
    Connect(ctx context.Context) error
    Close() error
    TestConnection(ctx context.Context) error
    Interrogate(ctx context.Context, commonAddress uint16) ([]PointValue, error)
    Listen(ctx context.Context, handler func(PointValue)) error
    Read(ctx context.Context, commonAddress uint16, ioa uint32) (PointValue, error)
    SendSingleCommand(ctx context.Context, commonAddress uint16, ioa uint32, value bool) error
    SendDoubleCommand(ctx context.Context, commonAddress uint16, ioa uint32, value DoubleCommandValue) error
    SendSetpoint(ctx context.Context, commonAddress uint16, ioa uint32, value SetpointValue) error
    SyncClock(ctx context.Context, commonAddress uint16, t time.Time) error
}
```

Suggested value model:

```go
type PointValue struct {
    Timestamp     time.Time
    CommonAddress uint16
    IOA           uint32
    Name          string
    Type          string
    Cause         string
    Value         any
    Quality       Quality
    RawTypeID     uint8
}
```

## Output contract

JSON output should be stable enough for scripts.

Example JSONL event:

```json
{"timestamp":"2026-05-18T12:34:56Z","common_address":1,"ioa":1001,"name":"active_power","type":"M_ME_NC_1","value":12.34,"unit":"MW","cause":"spontaneous","quality":{"invalid":false,"not_topical":false,"substituted":false,"blocked":false}}
```

Table output should be human-friendly:

```text
TIME                  CA  IOA   NAME          TYPE       VALUE  UNIT  CAUSE         QUALITY
2026-05-18 12:34:56   1   1001  active_power  M_ME_NC_1  12.34  MW    spontaneous   good
```

## Exit codes

```text
0  success
1  general error
2  config error
3  TCP connection error
4  IEC 104 session or STARTDT error
5  interrogation timeout
6  unsupported ASDU or type
7  command rejected
8  command timeout
9  output or formatting error
```

## Implementation milestones

### Milestone 1: skeleton

- Go module
- Makefile
- CLI root
- config loading
- config validation
- CI
- README bootstrap

### Milestone 2: read-only field tool

- `test-connection`
- `listen`
- `interrogate`
- point filters
- output formats

### Milestone 3: latest-value workflows

- latest-value cache
- `watch`
- stale markers
- configured point names and units

### Milestone 4: control operations

- dry-run safety model
- `command single`
- `command double`
- `setpoint`
- `clock-sync`

### Milestone 5: polish

- completions
- stable exit codes
- verbose and debug logging
- stronger README examples
- integration tests with local test server

## Non-goals for the first version

- IEC 101 support
- full IEC 62351 security support
- persistent local cache
- daemon mode
- GUI
- Modbus-style register mapping
- pretending IEC 104 has OPC UA-style node browsing

## Design warning

Do not design this as a generic SCADA framework yet. Build the smallest honest IEC 104 CLI first. Frameworks born before field usage usually grow antlers in the wrong places.
