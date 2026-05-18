package cli

import (
	"fmt"
	"io"
)

func writeCompletion(w io.Writer, shell string) error {
	switch shell {
	case "bash":
		_, err := fmt.Fprint(w, bashCompletion)
		return err
	case "zsh":
		_, err := fmt.Fprint(w, zshCompletion)
		return err
	default:
		return fmt.Errorf("unsupported shell %q; expected bash or zsh", shell)
	}
}

const bashCompletion = `# bash completion for iec-104-cli
_iec_104_cli()
{
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="help version validate-config generate-configs test-connection listen interrogate watch read command setpoint clock-sync completions"
    case "$prev" in
        iec-104-cli)
            COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
            return 0
            ;;
        command)
            COMPREPLY=( $(compgen -W "single double" -- "$cur") )
            return 0
            ;;
        setpoint)
            COMPREPLY=( $(compgen -W "normalized scaled float" -- "$cur") )
            return 0
            ;;
        completions)
            COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
            return 0
            ;;
    esac
}
complete -F _iec_104_cli iec-104-cli
`

const zshCompletion = `#compdef iec-104-cli
_iec_104_cli() {
  local -a commands
  commands=(
    'help:show help'
    'version:show version'
    'validate-config:validate local config'
    'generate-configs:generate example config files'
    'test-connection:run connection diagnostics'
    'listen:print incoming values'
    'interrogate:send general interrogation'
    'watch:print latest cached values'
    'read:read a specific IOA'
    'command:run control commands'
    'setpoint:run setpoint commands'
    'clock-sync:run clock synchronization'
    'completions:generate shell completions'
  )
  _describe 'command' commands
}
_iec_104_cli "$@"
`
