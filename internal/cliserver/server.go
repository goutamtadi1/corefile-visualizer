package cliserver

import (
	"io/fs"
	"net"
	"net/http"
	"strconv"
)

// Handler serves the embedded web app and a GET /corefile route returning the
// provided Corefile text as text/plain.
func Handler(app fs.FS, corefile string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/corefile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(corefile))
	})
	mux.Handle("/", http.FileServer(http.FS(app)))
	return mux
}

// ListenLocal binds 127.0.0.1 on the given port (0 = a random free port) and
// returns the listener so the caller can read the assigned address.
func ListenLocal(port int) (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
}
