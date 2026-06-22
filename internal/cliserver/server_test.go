package cliserver

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testApp() fstest.MapFS {
	return fstest.MapFS{
		"index.html":     {Data: []byte("<!doctype html><title>app</title>")},
		"assets/app.css": {Data: []byte("body{}")},
	}
}

func TestHandlerServesCorefile(t *testing.T) {
	srv := httptest.NewServer(Handler(testApp(), ". {\n  whoami\n}\n"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/corefile")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != ". {\n  whoami\n}\n" {
		t.Errorf("body = %q", string(body))
	}
}

func TestHandlerServesIndex(t *testing.T) {
	srv := httptest.NewServer(Handler(testApp(), "x"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<title>app</title>") {
		t.Errorf("index not served, body = %q", string(body))
	}
}

func TestHandlerServesAsset(t *testing.T) {
	srv := httptest.NewServer(Handler(testApp(), "x"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/assets/app.css")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestListenLocalRandomPort(t *testing.T) {
	ln, err := ListenLocal(0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if addr.Port == 0 {
		t.Fatal("expected a non-zero assigned port")
	}
}
