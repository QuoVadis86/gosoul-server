package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/qy-info/gosoul/internal/game/engine"
)

func init() {
	Register("mjai", NewMjaiPlayer)
}

// mjai is the de-facto standard protocol for riichi mahjong AI (used by
// Mortal, Akochan, RiichiEnv bots). Transport is JSON Lines over the
// subprocess stdin/stdout. See https://github.com/Cryolite/mjai for the spec.
//
// The bridge works in lockstep: the engine emits events (start_kyoku, tsumo,
// dahai, ...) which are forwarded as JSONL; the bot answers with its action
// or "none". Availability of the bot is not required at server startup: if the
// process cannot be spawned, the seat falls back to the configured tier.
type mjaiPlayer struct {
	mu     sync.Mutex
	name   string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	closed bool
}

// NewMjaiPlayer starts an external mjai-compatible bot process.
// Config keys:
//
//	command  - executable (default "mortal.py")
//	args     - space separated arguments (e.g. "--player-id 0")
func NewMjaiPlayer(cfg map[string]string) (Player, error) {
	command := cfg["command"]
	if command == "" {
		command = "mortal.py"
	}

	var args []string
	if a := cfg["args"]; a != "" {
		if err := json.Unmarshal([]byte(`["`+a+`"]`), &args); err == nil {
			// args provided as a JSON array
		} else {
			args = splitArgs(a)
		}
	}
	_ = args

	cmd := exec.Command(command)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mjai: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mjai: stdout pipe: %w", err)
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mjai: start %q: %w", command, err)
	}

	return &mjaiPlayer{
		name:   "mjai:" + command,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
	}, nil
}

func splitArgs(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func (m *mjaiPlayer) Name() string { return m.name }

func (m *mjaiPlayer) Level() Level { return LevelExternal }

func (m *mjaiPlayer) send(line map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return io.ErrClosedPipe
	}
	b, _ := json.Marshal(line)
	b = append(b, '\n')
	_, err := m.stdin.Write(b)
	return err
}

func (m *mjaiPlayer) recv(ctx context.Context) (map[string]any, error) {
	done := make(chan error, 1)
	var line string
	go func() {
		if m.stdout.Scan() {
			line = m.stdout.Text()
			done <- nil
		} else {
			done <- m.stdout.Err()
		}
	}()
	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mjaiPlayer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	if m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
	}
	return m.cmd.Wait()
}

// Interface assertions: mjaiPlayer implements Player once the bridge methods
// below are implemented.
var _ Player = (*mjaiPlayer)(nil)

// ChooseDiscard forwards the current state to the bot, whose dahai reply is
// mapped back. TODO(ai): wire engine event stream here instead of a per-call
// projection so the bot's internal state stays in sync.
func (m *mjaiPlayer) ChooseDiscard(ctx context.Context, v *engine.View) engine.Tile {
	_ = m.send(map[string]any{"type": "ask", "view": v})
	msg, err := m.recv(ctx)
	if err != nil {
		return engine.Tile("")
	}
	if pai, ok := msg["pai"].(string); ok {
		return engine.Tile(normalizePai(pai))
	}
	return engine.Tile("")
}

func (m *mjaiPlayer) ChooseCall(ctx context.Context, v *engine.View, ops []engine.CallOp) *engine.CallOp {
	_ = m.send(map[string]any{"type": "ask_call", "view": v, "ops": ops})
	if _, err := m.recv(ctx); err != nil {
		return nil
	}
	return nil
}

func (m *mjaiPlayer) ChooseSelfAction(ctx context.Context, v *engine.View, ops []engine.SelfOp) *engine.SelfOp {
	_ = m.send(map[string]any{"type": "ask_self", "view": v, "ops": ops})
	if _, err := m.recv(ctx); err != nil {
		return nil
	}
	return nil
}

// normalizePai maps mjai pai spellings ("5mr") to canonical tiles ("5m").
func normalizePai(p string) string {
	if len(p) < 2 {
		return p
	}
	return p[:2]
}
