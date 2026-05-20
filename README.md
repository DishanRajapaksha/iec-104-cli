# IEC 104 CLI

`iec-104-cli` is a script-friendly command-line client for IEC 60870-5-104.

## At a Glance

| Task | Command |
|---|---|
| Validate local config | `iec-104-cli validate-config` |
| Test TCP and STARTDT | `iec-104-cli test-connection` |
| Listen for incoming values | `iec-104-cli listen --duration 10s --format jsonl` |
| Run general interrogation | `iec-104-cli interrogate --common-address 1 --format table` |
| Watch latest cached value | `iec-104-cli watch --point active_power --interval 1s` |
| Read one IOA | `iec-104-cli read --ioa 1001` |
| Dry-run a single command | `iec-104-cli command single --ioa 1000 --value on` |
| Execute a single command | `iec-104-cli command single --ioa 1000 --value on --yes` |
| Dry-run a setpoint | `iec-104-cli setpoint float --ioa 2002 --value 12.5` |
| Dry-run clock sync | `iec-104-cli clock-sync` |
| JSON Lines output | `iec-104-cli listen --duration 10s --format jsonl` |
| CSV output | `iec-104-cli interrogate --format csv` |

## Install

Build from source:

```bash
make build
./bin/iec-104-cli help
```

Build a container image:

```bash
make docker-build
docker run --rm iec-104-cli:latest help
```

Build a Debian package:

```bash
make deb
sudo apt install ./dist/iec-104-cli_0.1.0_amd64.deb
```

If multiple local users work in the same checkout, fix shared write permissions:

```bash
sudo scripts/fix-shared-permissions.sh
```

## First Run

Create `config.yaml`, edit the connection and points, then validate it:

```bash
iec-104-cli init-config
iec-104-cli validate-config
```

Config files default to `config.yaml`. Use `--config site.yaml` to select another file, and `--profile plant-a` to select a named profile from that file.

## Config Example

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

cache:
  enabled: true
  path: .iec-104-cli/cache.json

points:
  - name: active_power
    ioa: 1001
    type: float
    unit: MW

default_profile: plant-a
profiles:
  plant-a:
    connection:
      host: 192.0.2.10
    iec104:
      common_address: 7
    output:
      format: json
```

## Commands

Validate local config without connecting:

```bash
iec-104-cli validate-config --config config.example.yaml
iec-104-cli validate-config --config config.yaml --profile plant-a
```

Test TCP and IEC 104 STARTDT:

```bash
iec-104-cli test-connection --host 127.0.0.1 --port 2404 --timeout 5s
iec-104-cli status --host 127.0.0.1 --port 2404 --timeout 5s
```

`status` is currently an alias for `test-connection`.

Listen for spontaneous or incoming values:

```bash
iec-104-cli listen --duration 10s --format jsonl
```

Run general interrogation:

```bash
iec-104-cli interrogate --common-address 1 --format table
iec-104-cli interrogate --point active_power --format json
```

Watch the latest value cache:

```bash
iec-104-cli watch --point active_power --interval 1s --format jsonl
iec-104-cli watch --cache .iec-104-cli/cache.json
iec-104-cli watch --no-cache
```

Read a specific IOA:

```bash
iec-104-cli read --ioa 1001 --format json
```

IEC 104 read is not equivalent to OPC UA read. Many devices prefer interrogation or spontaneous updates and may not answer read commands.

## Command Safety

Control commands are dry-run by default. Use `--yes` to send a real command.

```bash
iec-104-cli command single --ioa 1000 --value on
iec-104-cli command single --ioa 1000 --value on --dry-run
iec-104-cli command single --ioa 1000 --value on --yes
iec-104-cli command double --ioa 1001 --value close --dry-run
```

Setpoints use the same safety model:

```bash
iec-104-cli setpoint normalized --ioa 2000 --value 0.5 --dry-run
iec-104-cli setpoint scaled --ioa 2001 --value 42 --dry-run
iec-104-cli setpoint float --ioa 2002 --value 12.5 --dry-run
```

Clock sync is also dry-run by default:

```bash
iec-104-cli clock-sync --dry-run
iec-104-cli clock-sync --time 2026-05-18T12:00:00Z --dry-run
```

## Output Formats

Most read-only commands support:

```text
table
text
json
jsonl
csv
```

Verbose and debug logs go to stderr so JSON and JSONL stdout remain script-friendly.

## Shell Completions

```bash
iec-104-cli completions bash > /tmp/iec-104-cli.bash
iec-104-cli completions zsh > /tmp/_iec-104-cli
```

## Example Config Files

Generate multi-file examples, including point CSV/YAML examples:

```bash
iec-104-cli generate-configs --dir examples
```

## Exit Codes

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

## Troubleshooting

Use `validate-config` first to catch local YAML and point definition problems.

Use `test-connection --verbose` to separate TCP connection failures from IEC 104 STARTDT/session failures.

Use `listen --debug --format jsonl` when checking spontaneous updates. Debug logs are written to stderr.
