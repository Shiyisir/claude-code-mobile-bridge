package launcher

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/shiyisir/claude-proxy/internal/config"
	"github.com/shiyisir/claude-proxy/internal/eventbus"
	"github.com/shiyisir/claude-proxy/internal/logging"
	"github.com/shiyisir/claude-proxy/internal/parser"
	"github.com/shiyisir/claude-proxy/internal/probe"
	"github.com/shiyisir/claude-proxy/internal/security"
	"github.com/shiyisir/claude-proxy/internal/stream"
	"github.com/shiyisir/claude-proxy/internal/writer"
	"github.com/shiyisir/claude-proxy/internal/ws"
)

// Start launches the real Claude with transparent forwarding + probe + parser + WebSocket.
func Start(cfg *config.Config, args []string) error {
	// Phase 0: capture probe
	rec := probe.Capture(cfg, args)
	if err := probe.Write(rec); err != nil {
		logging.Warn("probe: write failed, continuing", "error", err)
	}

	// Phase 1: event writer (async, buffered, file-based)
	ew, err := writer.NewEventWriter()
	if err != nil {
		logging.Warn("launcher: event writer failed, continuing without events", "error", err)
		ew = nil
	}

	// Phase 2: event bus + WebSocket server
	var bus *eventbus.Bus
	var wsServer *ws.Server
	if cfg.EnableWS {
		bus = eventbus.New(cfg.MaxWSQueue)

		wsCfg := ws.Config{
			Host:       cfg.WSHost,
			Port:       cfg.WSPort,
			Token:      "", // set below after token generation
			MaxClients: cfg.MaxWSClients,
			MaxWSQueue: cfg.MaxWSQueue,
			EnableWS:   true,
		}

		token, err := security.GenerateToken()
		if err != nil {
			logging.Warn("launcher: token generation failed", "error", err)
		} else {
			wsCfg.Token = token
			wsServer = ws.NewServer(wsCfg, bus)
			if err := wsServer.Start(); err != nil {
				logging.Warn("launcher: ws server start failed", "error", err)
				wsServer = nil
			} else {
				// Only save token AFTER successful start — avoids stale files
				tokenPath := cfg.WSTokenFile
				if tokenPath == "" {
					tokenPath = config.DefaultTokenPath()
				}
				security.SaveToken(tokenPath, token)
				logging.Info("launcher: ws server ready", "port", cfg.WSPort, "clients", cfg.MaxWSClients)
			}
		}
	}

	// Prepare real claude command
	cmd := exec.Command(cfg.RealBin, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin

	var stdoutSample, stderrSample []byte
	var stdoutSidecar *stream.Sidecar

	// Only parse stdout if enabled
	if cfg.EnableJSON && (ew != nil || bus != nil) {
		pCfg := parser.DefaultConfig()

		// Fan-out callback: push to writer AND eventbus
		onEvent := func(event *parser.NormalizedEvent) {
			if ew != nil {
				ew.Write(event)
			}
			if bus != nil {
				data, err := json.Marshal(event)
				if err == nil {
					bus.Publish(&eventbus.Event{
						SessionID: event.SessionID,
						Data:      data,
					})
				}
			}
		}

		stdoutSidecar = stream.NewSidecar(1024)
		p := parser.New(pCfg, "stdout", onEvent)
		go p.Run(stdoutSidecar)
	}

	// stdout pipe
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logging.Error("launcher: stdout pipe failed, falling back", "error", err)
		Fallback(args)
		return nil
	}

	// stderr pipe (no sidecar)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logging.Error("launcher: stderr pipe failed, falling back", "error", err)
		Fallback(args)
		return nil
	}

	if err := cmd.Start(); err != nil {
		logging.Error("launcher: start real claude failed", "error", err)
		return err
	}

	// Copy stdout/stderr in parallel
	done := make(chan struct{}, 2)
	go func() {
		stream.Tee(os.Stdout, stdoutPipe, &stdoutSample, stdoutSidecar)
		done <- struct{}{}
	}()
	go func() {
		stream.Tee(os.Stderr, stderrPipe, &stderrSample, nil)
		done <- struct{}{}
	}()

	// Wait for process first
	err = cmd.Wait()

	// Drain remaining pipe data
	<-done
	<-done

	// Log stats
	if ew != nil {
		written, noSid := ew.Stats()
		logging.Info("launcher: event writer stats", "written", written, "no_session_id", noSid)
		ew.Close()
	}
	if bus != nil {
		subs, dropped := bus.Stats()
		logging.Info("launcher: eventbus stats", "subscribers", subs, "dropped", dropped)
	}
	if wsServer != nil {
		logging.Info("launcher: ws server stats", "clients", wsServer.ClientCount())
		wsServer.Stop()
	}

	// Attach samples to probe
	probe.AttachSample(rec, stdoutSample, stderrSample)

	// Preserve exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			logging.Debug("real claude exited", "code", code)
			os.Exit(code)
		}
		return err
	}

	return nil
}
