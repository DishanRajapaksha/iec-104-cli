# IEC 104 CLI Agent Task List

This file breaks the `iec-104-cli` implementation into agent-sized tasks.

Each task should be small enough to complete in a focused pull request. Prefer one task per PR unless the tasks are tightly coupled. Do not build control commands before the read-only path works.

## Task status key

```text
[ ] not started
[x] done
[~] in progress
[!] blocked
```

## Task 1: Scaffold Go module and CLI entrypoint

Status: [x]

Create the basic Go project structure.

Files likely touched:

```text
go.mod
go.sum
main.go
Makefile
internal/cli/root.go
internal/exitcode/exitcode.go
.github/workflows/ci.yml
```

Requirements:

- Module path must be `github.com/DishanRajapaksha/iec-104-cli`.
- Binary name must be `iec-104-cli`.
- `main.go` should delegate to `internal/cli`.
- Add `make fmt`, `make test`, `make build`, and `make clean`.
- Add GitHub Actions CI for `go test ./...`.

Acceptance:

```bash
go test ./...
make build
./bin/iec-104-cli help
```

## Task 2: Add root command, shared flags, and help text

Status: [x]

Implement the CLI root command and global flags.

Global flags:

```text
--config config.yaml
--profile string
--format table|text|json|jsonl
--timeout duration
--verbose
--debug
```

Requirements:

- Default config path must be `config.yaml`.
- Help output should show available subcommands.
- Unknown commands should fail cleanly.
- Invalid output format should return config error code.

Acceptance:

```bash
iec-104-cli help
iec-104-cli --help
iec-104-cli --format json help
iec-104-cli --format nonsense validate-config
```

## Task 3: Add config model and YAML loader

Status: [x]

Implement config parsing and merging.

Files likely touched:

```text
internal/config/config.go
internal/config/config_test.go
config.example.yaml
```

Initial config shape:

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
```

Requirements:

- Load from `config.yaml` by default when present.
- Allow missing config only for `help` and commands that can run without config.
- Allow `--config` override.
- CLI flags must override file config.
- Keep config structs independent from CLI structs.

Acceptance:

```bash
iec-104-cli validate-config
iec-104-cli validate-config --config config.example.yaml
iec-104-cli test-connection --host 127.0.0.1 --port 2404
```

## Task 4: Add `validate-config`

Status: [x]

Validate local config without connecting to a server.

Validation rules:

- host is required for connection commands
- port must be between 1 and 65535
- timeout must be positive
- reconnect interval must be positive when reconnect is enabled
- common address must be valid for IEC 104 usage
- originator address must be valid
- interrogation qualifier must be valid
- point names must be unique
- point IOAs must be valid
- point types must be recognised
- output format must be recognised

Accepted point types for the first version:

```text
single_point
double_point
normalized
scaled
float
integrated_total
```

Acceptance:

```bash
iec-104-cli validate-config --config config.example.yaml
```

Expected result: exit code 0 and a short success message.

## Task 5: Add output formatting package

Status: [x]

Implement output helpers for table, text, JSON, and JSONL.

Files likely touched:

```text
internal/cli/output.go
internal/iec104/types.go
```

Requirements:

- JSON field names must be stable.
- JSONL must output one event per line.
- Table output must be readable in a terminal.
- Text output must be minimal and script-friendly.
- Output code must not directly depend on the third-party IEC 104 library.

Acceptance:

- Unit tests cover table, text, JSON, and JSONL for a sample point value.
- JSON output is valid JSON.
- JSONL output is valid one-object-per-line JSON.

## Task 6: Add IEC 104 internal domain types

Status: [x]

Create protocol-independent local types.

Files likely touched:

```text
internal/iec104/types.go
internal/iec104/errors.go
```

Suggested types:

```go
type PointValue struct {
    Timestamp     time.Time
    CommonAddress uint16
    IOA           uint32
    Name          string
    Type          string
    Cause         string
    Value         any
    Unit          string
    Quality       Quality
    RawTypeID     uint8
}

type Quality struct {
    Invalid    bool
    NotTopical bool
    Substituted bool
    Blocked    bool
}
```

Requirements:

- Keep types serialisable to JSON.
- Avoid `any` where a stronger type is obvious, but do not over-engineer before backend mapping is known.
- Include helpers for display quality such as `good`, `invalid`, `blocked`, or combined flags.

Acceptance:

```bash
go test ./...
```

## Task 7: Add `wendy512/iec104` adapter spike

Status: [x]

Add the dependency and wrap it behind a local interface.

Files likely touched:

```text
go.mod
go.sum
internal/iec104/client.go
internal/iec104/adapter_wendy.go
```

Local interface:

```go
type Client interface {
    Connect(ctx context.Context) error
    Close() error
    TestConnection(ctx context.Context) error
    Interrogate(ctx context.Context, commonAddress uint16) ([]PointValue, error)
    Listen(ctx context.Context, handler func(PointValue)) error
}
```

Requirements:

- Third-party client types must not appear in `internal/cli`.
- Adapter must honour context cancellation where possible.
- Adapter must map connection failures to local error types.
- Adapter must include enough comments to explain callback bridging.

Acceptance:

```bash
go test ./...
```

## Task 8: Add `test-connection`

Status: [x]

Implement connection testing.

Command:

```bash
iec-104-cli test-connection
```

Requirements:

- Open TCP connection.
- Start IEC 104 data transfer.
- Print host, port, TCP result, IEC 104 result, and final status.
- Respect timeout.
- Return clear exit codes.

Acceptance:

```bash
iec-104-cli test-connection --host 127.0.0.1 --port 2404 --timeout 5s
```

Expected behaviour:

- connection refused returns TCP connection error code
- timeout returns TCP connection error code
- protocol start failure returns IEC 104 session error code

## Task 9: Add `listen`

Status: [x]

Implement passive event listening.

Command:

```bash
iec-104-cli listen
```

Useful flags:

```text
--duration 30s
--common-address 1
--ioa 1001
--point active_power
--format jsonl
```

Requirements:

- Connect and print incoming point values.
- Stop on Ctrl+C.
- Stop after `--duration` when provided.
- Filter by common address, IOA, or configured point name.
- Output in selected format.

Acceptance:

```bash
iec-104-cli listen --duration 10s --format jsonl
```

## Task 10: Add `interrogate`

Status: [ ]

Implement general interrogation.

Command:

```bash
iec-104-cli interrogate
```

Useful flags:

```text
--common-address 1
--timeout 10s
--ioa 1001
--point active_power
--format json
```

Requirements:

- Send general interrogation command.
- Collect values until completion or timeout.
- Map IOAs to configured point names and units.
- Support filters.
- Return interrogation timeout exit code when no completion is received.

Acceptance:

```bash
iec-104-cli interrogate --common-address 1 --format table
iec-104-cli interrogate --point active_power --format json
```

## Task 11: Add point lookup and filters

Status: [ ]

Add reusable point lookup and filtering.

Files likely touched:

```text
internal/config/config.go
internal/iec104/types.go
internal/cli/filter.go
```

Requirements:

- Filter by point name.
- Filter by IOA.
- Filter by common address.
- Add configured name and unit to emitted values.
- Handle unknown point names with config error.

Acceptance:

```bash
iec-104-cli interrogate --point active_power
iec-104-cli listen --ioa 1001
```

## Task 12: Add latest-value cache and `watch`

Status: [ ]

Implement latest-value cache and watch command.

Command:

```bash
iec-104-cli watch
```

Useful flags:

```text
--interval 1s
--stale-after 30s
--point active_power
--ioa 1001
--format jsonl
```

Requirements:

- Maintain latest value by common address and IOA.
- Print matching values on interval.
- Mark stale values after configured threshold.
- Do not repeatedly send interrogation unless a future flag explicitly asks for it.

Acceptance:

```bash
iec-104-cli watch --point active_power --interval 1s --format jsonl
```

## Task 13: Add `read`

Status: [ ]

Implement IEC 104 read command for a specific IOA.

Command:

```bash
iec-104-cli read --ioa 1001
```

Requirements:

- Send IEC 104 read command.
- Wait for matching response or timeout.
- Explain in README that many devices may prefer interrogation or spontaneous updates.
- Return clear timeout error when unsupported by server.

Acceptance:

```bash
iec-104-cli read --ioa 1001 --format json
```

## Task 14: Add dry-run safety model for control commands

Status: [ ]

Create shared safety handling for control operations.

Rules:

- Dry-run by default.
- Require `--yes` to send a real command.
- Refuse real commands in non-interactive mode without `--yes`.
- Print common address, IOA, type, value, and qualifier before execution.
- Support `--dry-run` explicitly.

Acceptance:

```bash
iec-104-cli command single --ioa 1000 --value on
iec-104-cli command single --ioa 1000 --value on --dry-run
iec-104-cli command single --ioa 1000 --value on --yes
```

The first two must not send anything.

## Task 15: Add `command single`

Status: [ ]

Implement single command.

Command:

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

Requirements:

- Use shared dry-run safety model.
- Parse value clearly.
- Wait for command response when supported.
- Return command rejected or timeout exit codes where possible.

Acceptance:

```bash
iec-104-cli command single --ioa 1000 --value on --dry-run
```

## Task 16: Add `command double`

Status: [ ]

Implement double command.

Command:

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

Acceptance:

```bash
iec-104-cli command double --ioa 1001 --value close --dry-run
```

## Task 17: Add setpoint commands

Status: [ ]

Implement setpoint commands.

Commands:

```bash
iec-104-cli setpoint normalized --ioa 2000 --value 0.5 --yes
iec-104-cli setpoint scaled --ioa 2001 --value 42 --yes
iec-104-cli setpoint float --ioa 2002 --value 12.5 --yes
```

Requirements:

- Use shared dry-run safety model.
- Validate numeric ranges.
- Return clear errors for unsupported or rejected setpoints.

Acceptance:

```bash
iec-104-cli setpoint float --ioa 2002 --value 12.5 --dry-run
```

## Task 18: Add `clock-sync`

Status: [ ]

Implement clock synchronisation command.

Command:

```bash
iec-104-cli clock-sync --yes
```

Requirements:

- Dry-run by default.
- Require `--yes` for actual execution.
- Default to current system time.
- Allow `--time` override with RFC3339 timestamp.
- Print the exact time that would be sent.

Acceptance:

```bash
iec-104-cli clock-sync --dry-run
iec-104-cli clock-sync --time 2026-05-18T12:00:00Z --dry-run
```

## Task 19: Add verbose and debug logging

Status: [ ]

Implement logging flags.

Requirements:

- `--verbose` prints high-level connection and command decisions.
- `--debug` prints protocol-level summaries.
- Logs must not corrupt JSON or JSONL stdout. Send logs to stderr.
- Do not print unreadable byte soup by default.

Acceptance:

```bash
iec-104-cli test-connection --verbose
iec-104-cli listen --debug --format jsonl
```

## Task 20: Add shell completions

Status: [ ]

Generate shell completions.

Commands:

```bash
iec-104-cli completions bash
iec-104-cli completions zsh
```

Acceptance:

```bash
iec-104-cli completions bash > /tmp/iec-104-cli.bash
iec-104-cli completions zsh > /tmp/_iec-104-cli
```

## Task 21: Add stable exit codes

Status: [ ]

Implement and document stable exit codes.

Exit codes:

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

Requirements:

- Internal errors should map to these codes.
- README must document the codes.
- Tests should cover representative mappings.

Acceptance:

```bash
iec-104-cli validate-config --config invalid.yaml
```

Expected result: exit code 2.

## Task 22: Add README documentation

Status: [ ]

Expand README into user-facing documentation.

Sections:

- overview
- install
- build from source
- first run
- config example
- validate config
- test connection
- listen
- interrogate
- watch
- read
- command safety
- single and double commands
- setpoints
- clock sync
- output formats
- exit codes
- troubleshooting

Acceptance:

- README examples match real commands.
- README does not claim unsupported behaviour.
- README clearly explains that IEC 104 read is not equivalent to OPC UA read.

## Task 23: Add integration tests with local server

Status: [ ]

Use the backend library server package or a lightweight fake server to test behaviour.

Test cases:

- client connects to local server
- `test-connection` succeeds
- interrogation returns values
- listen receives spontaneous values
- single command reaches server handler
- command dry-run sends nothing

Acceptance:

```bash
go test ./...
```

Tests must not require external network access.

## Task 24: Add release build workflow

Status: [ ]

Add cross-platform release builds later, after the CLI stabilises.

Targets:

```text
linux-amd64
linux-arm64
darwin-amd64
darwin-arm64
windows-amd64
```

Requirements:

- Build artifacts should use the binary name `iec-104-cli`.
- Release workflow should run tests first.
- Do not add release automation before the first working CLI milestone.

Acceptance:

- Manual workflow can build all target binaries.

## Task 25: Field hardening backlog

Status: [ ]

Do after the basic CLI works.

Possible additions:

- reconnect loop for `listen`
- reconnect loop for `watch`
- protocol frame debug dump behind an explicit flag
- TLS or IEC 62351 research spike
- CSV output
- point file import
- generated example configs
- Docker image
- Debian package
- Homebrew formula
- GitHub release checksums

Acceptance:

- Each item should become its own task before implementation.

## Recommended PR order

1. Task 1
2. Task 2
3. Task 3
4. Task 4
5. Task 5
6. Task 6
7. Task 7
8. Task 8
9. Task 9
10. Task 10
11. Task 11
12. Task 12
13. Task 14
14. Task 15
15. Task 16
16. Task 17
17. Task 18
18. Task 19
19. Task 20
20. Task 21
21. Task 22
22. Task 23
23. Task 24
24. Task 25

## Agent instruction

Before starting any task:

1. Read `PLAN.md`.
2. Read this file.
3. Check the current repository state.
4. Keep the change small.
5. Run `go test ./...` before opening a PR.
6. Do not add control-command behaviour without dry-run safety.
7. Do not import third-party IEC 104 packages directly in the CLI layer.
