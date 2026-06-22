// Command corefile-visualizer reads a CoreDNS Corefile (from a piped stdin or a
// file argument), serves the embedded web visualization from a local HTTP
// server with that content pre-loaded, opens the browser, and runs until Ctrl-C.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gtadi/corefile-visualizer/internal/cliserver"
	"github.com/gtadi/corefile-visualizer/internal/webui"
)

// openBrowser is a variable so tests can stub it.
var openBrowser = openBrowserDefault

func main() {
	noOpen := flag.Bool("no-open", false, "start the server but do not open a browser")
	port := flag.Int("port", 0, "port to listen on (0 = a random free port)")
	flag.Parse()

	fi, _ := os.Stdin.Stat()
	stdinIsPipe := (fi.Mode() & os.ModeCharDevice) == 0

	content, err := cliserver.ReadCorefile(os.Stdin, stdinIsPipe, flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, "usage: corefile-visualizer [--no-open] [--port N] [FILE]  (or pipe a Corefile via stdin)")
		os.Exit(2)
	}

	app, err := webui.FS()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: embedded web app unavailable:", err)
		os.Exit(1)
	}

	ln, err := cliserver.ListenLocal(*port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot listen:", err)
		os.Exit(1)
	}
	url := fmt.Sprintf("http://%s/", ln.Addr().String())

	srv := &http.Server{
		Handler:           cliserver.Handler(app, content),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "server error:", err)
		}
	}()

	fmt.Println("CoreDNS Corefile Visualizer serving at", url)
	if *noOpen {
		fmt.Println("(--no-open) open the URL above in your browser")
	} else if err := openBrowser(url); err != nil {
		fmt.Fprintln(os.Stderr, "could not open browser automatically:", err)
	}
	fmt.Println("Press Ctrl-C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nshutting down")
	_ = srv.Close()
}

func openBrowserDefault(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	return exec.Command(name, args...).Start()
}
