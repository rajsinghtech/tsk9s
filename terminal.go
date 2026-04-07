package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func handleTerminal(kubeconfigPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // tailnet-internal only
		})
		if err != nil {
			log.Printf("ws accept: %v", err)
			return
		}
		defer conn.CloseNow()

		ctx := r.Context()

		k9sPath, err := exec.LookPath("k9s")
		if err != nil {
			log.Printf("k9s not found in PATH: %v", err)
			conn.Close(websocket.StatusInternalError, "k9s not found in PATH")
			return
		}

		cmd := exec.CommandContext(ctx, k9sPath)
		cmd.Env = append(os.Environ(),
			"TERM=xterm-256color",
			"KUBECONFIG="+kubeconfigPath,
		)

		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Printf("pty.Start: %v", err)
			conn.Close(websocket.StatusInternalError, "failed to start k9s")
			return
		}
		defer ptmx.Close()

		// Set initial size
		pty.Setsize(ptmx, &pty.Winsize{Cols: 120, Rows: 40})

		var wg sync.WaitGroup

		// PTY stdout -> WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 32*1024)
			for {
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("pty read: %v", err)
					}
					return
				}
				if err := conn.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
					return
				}
			}
		}()

		// WebSocket -> PTY stdin (or resize)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, data, err := conn.Read(ctx)
				if err != nil {
					return
				}

				// Try to parse as resize message
				var rm resizeMsg
				if json.Unmarshal(data, &rm) == nil && rm.Type == "resize" && rm.Cols > 0 && rm.Rows > 0 {
					pty.Setsize(ptmx, &pty.Winsize{Cols: rm.Cols, Rows: rm.Rows})
					continue
				}

				// Otherwise write as input
				if _, err := ptmx.Write(data); err != nil {
					return
				}
			}
		}()

		// Wait for k9s to exit
		if err := cmd.Wait(); err != nil {
			log.Printf("k9s exited: %v", err)
		}
		conn.Close(websocket.StatusNormalClosure, "k9s exited")
		wg.Wait()
	}
}
