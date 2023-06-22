package supply

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Command struct {
	argv []string
}

func NewCommandSupply(cmd string) (*Command, error) {
	var argv []string
	if strings.HasPrefix(cmd, "!") {
		cmd = strings.TrimPrefix(cmd, "!")
		if cmd == "" {
			return nil, fmt.Errorf("no command given")
		}

		argv = []string{"sh", "-c", cmd}
	} else {
		argv = strings.Fields(cmd)
		if len(argv) == 0 {
			return nil, fmt.Errorf("no command given")
		}
	}

	return &Command{argv}, nil
}

func (s *Command) Supply(ctx context.Context) ([]map[string]string, error) {
	cmd := exec.CommandContext(ctx, s.argv[0], s.argv[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var data []map[string]string
	err = json.Unmarshal(out, &data)
	return data, err
}

func (s *Command) ShouldCache() bool {
	return true
}
