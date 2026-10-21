package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	liberrors "rapid/lib/errors"
)

// resolveTimeout is the whole-request budget (Python: 8000 ms). A var so tests
// can shorten it instead of sleeping for real.
var resolveTimeout = 8 * time.Second

// terminateGrace mirrors the Python terminate timeout: after the context kills
// the child, io pipes get at most this long to close.
const terminateGrace = 2 * time.Second

// runResolve spawns the plugin, sends one rapid.resolve, and reads one answer.
// One process per request; no ping, no pooling.
// ponytail: no rapid.ping health gate — a failed resolve already covers a dead
// plugin, and the extra process round-trip buys nothing for os/exec.
func runResolve(ctx context.Context, m Manifest, uri string, options map[string]any) ([]ResolvedItem, error) {
	command, err := pluginCommand(m)
	if err != nil {
		return nil, err
	}

	req := ResolveRequest(1, uri, options)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, m.Args...)
	cmd.Dir = m.Dir
	cmd.WaitDelay = terminateGrace
	cmd.Stdin = bytes.NewReader(append(payload, '\n'))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	line, readErr := readLine(ctx, stdout)
	cancel()

	cmd.Wait() // reap; the error is already classified by readErr/ctx
	if readErr != nil {
		return nil, readErr
	}

	return decodeResolve(req.ID, line)
}

type lineResult struct {
	line string
	err  error
}

func readLine(ctx context.Context, r io.Reader) (string, error) {
	lines := make(chan lineResult, 1)

	go func() {
		line, err := bufio.NewReader(r).ReadString('\n')
		lines <- lineResult{line, err}
	}()

	select {
	case <-ctx.Done():
		return "", liberrors.NewTransportError("plugin response timeout", ctx.Err())
	case got := <-lines:
		if got.err != nil {
			if got.line == "" {
				return "", liberrors.NewTransportError("plugin returned nothing", got.err)
			}
			return "", liberrors.NewTransportError("plugin response truncated", got.err)
		}
		return strings.TrimSpace(got.line), nil
	}
}

func decodeResolve(id int, line string) ([]ResolvedItem, error) {
	var resp ResolveResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return nil, err
	}
	if resp.ID != id {
		return nil, liberrors.NewJsonRpcError(-32600, fmt.Sprintf("response id mismatch: got %d want %d", resp.ID, id))
	}
	if resp.Error != nil {
		return nil, liberrors.NewJsonRpcError(resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		return nil, liberrors.NewJsonRpcError(-32603, "resolve returned invalid result")
	}

	return resp.Result.Items, nil
}

// pluginCommand resolves transport.command. A command with a path separator is
// relative to the manifest dir; a bare name tries the manifest dir first, then
// PATH (so "plugin.py" runs from the dir and "python3" comes from PATH).
func pluginCommand(m Manifest) (string, error) {
	if strings.ContainsAny(m.Command, `/\`) {
		return filepath.Join(m.Dir, m.Command), nil
	}

	local := filepath.Join(m.Dir, m.Command)
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return local, nil
	}

	path, err := exec.LookPath(m.Command)
	if err != nil {
		return "", err
	}
	return path, nil
}
