// Package plugins holds the CoreDNS plugin execution order and builds a server
// block's request-flow chain. The order is taken verbatim from CoreDNS's
// plugin.cfg at v1.12.0:
// https://github.com/coredns/coredns/blob/v1.12.0/plugin.cfg
package plugins

import (
	"sort"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

// Order is the CoreDNS plugin.cfg execution order (index = priority). Sourced
// verbatim from CoreDNS v1.12.0; do not reorder.
// Callers must not mutate Order; Rank is precomputed from it at init time.
var Order = []string{
	"root", "metadata", "geoip", "cancel", "tls", "timeouts", "multisocket",
	"reload", "nsid", "bufsize", "bind", "debug", "trace", "ready", "health",
	"pprof", "prometheus", "errors", "log", "dnstap", "local", "dns64", "acl",
	"any", "chaos", "loadbalance", "tsig", "cache", "rewrite", "header", "dnssec",
	"autopath", "minimal", "template", "transfer", "hosts", "route53", "azure",
	"clouddns", "k8s_external", "kubernetes", "file", "auto", "secondary", "etcd",
	"loop", "forward", "grpc", "erratic", "whoami", "on", "sign", "view",
}

var rankByName = func() map[string]int {
	m := make(map[string]int, len(Order))
	for i, name := range Order {
		m[name] = i
	}
	return m
}()

// Rank returns the plugin's index in Order and whether it is a known plugin.
func Rank(name string) (int, bool) {
	i, ok := rankByName[name]
	return i, ok
}

// BuildFlow returns the request-execution chain for a server block's top-level
// directives: distinct plugin names (first occurrence wins), known plugins
// sorted by plugin.cfg rank, then unknown plugins appended in declaration order.
// The result is always non-nil.
func BuildFlow(directives []model.Directive) []model.FlowStep {
	seen := map[string]bool{}
	var known []string
	var unknown []string
	for _, d := range directives {
		if seen[d.Name] {
			continue
		}
		seen[d.Name] = true
		if _, ok := Rank(d.Name); ok {
			known = append(known, d.Name)
		} else {
			unknown = append(unknown, d.Name)
		}
	}
	sort.SliceStable(known, func(i, j int) bool {
		ri, _ := Rank(known[i])
		rj, _ := Rank(known[j])
		return ri < rj
	})

	flow := make([]model.FlowStep, 0, len(known)+len(unknown))
	for _, n := range known {
		flow = append(flow, model.FlowStep{Name: n, Known: true})
	}
	for _, n := range unknown {
		flow = append(flow, model.FlowStep{Name: n, Known: false})
	}
	return flow
}

// Meta is reference metadata for a plugin: a one-line summary (from coredns.io)
// and a documentation URL (empty when the plugin has no coredns.io page).
type Meta struct {
	Summary string `json:"summary"`
	DocURL  string `json:"docUrl"`
}

func doc(name string) string { return "https://coredns.io/plugins/" + name + "/" }

var meta = map[string]Meta{
	"root":         {"specifies the root of where to find zone files.", doc("root")},
	"metadata":     {"enables a metadata collector.", doc("metadata")},
	"geoip":        {"looks up .mmdb databases using the client IP and adds geoip data to the request context.", doc("geoip")},
	"cancel":       {"cancels a request's context after 5001 milliseconds.", doc("cancel")},
	"tls":          {"configures the server certificates for the TLS, gRPC and DoH servers.", doc("tls")},
	"timeouts":     {"configures server read, write and idle timeouts for TCP, TLS, DoH and DoQ servers.", doc("timeouts")},
	"multisocket":  {"starts multiple servers that listen on one port.", doc("multisocket")},
	"reload":       {"allows automatic reload of a changed Corefile.", doc("reload")},
	"nsid":         {"adds an identifier of this server to each reply.", doc("nsid")},
	"bufsize":      {"limits the EDNS0 buffer size to prevent IP fragmentation.", doc("bufsize")},
	"bind":         {"overrides the host to which the server should bind.", doc("bind")},
	"debug":        {"disables automatic crash recovery so you get a stack trace.", doc("debug")},
	"trace":        {"enables OpenTracing-based tracing of DNS requests through the plugin chain.", doc("trace")},
	"ready":        {"enables a readiness check HTTP endpoint.", doc("ready")},
	"health":       {"enables a health check endpoint.", doc("health")},
	"pprof":        {"publishes runtime profiling data at endpoints under /debug/pprof.", doc("pprof")},
	"prometheus":   {"enables Prometheus metrics.", doc("prometheus")},
	"errors":       {"enables error logging.", doc("errors")},
	"log":          {"enables query logging to standard output.", doc("log")},
	"dnstap":       {"enables logging to dnstap.", doc("dnstap")},
	"local":        {"responds to local names.", doc("local")},
	"dns64":        {"enables the DNS64 IPv6 transition mechanism.", doc("dns64")},
	"acl":          {"enforces access control policies on the source IP.", doc("acl")},
	"any":          {"gives a minimal response to ANY queries.", doc("any")},
	"chaos":        {"responds to TXT queries in the CH class.", doc("chaos")},
	"loadbalance":  {"randomizes the order of A, AAAA and MX records.", doc("loadbalance")},
	"tsig":         {"defines TSIG keys and validates/signs TSIG requests and responses.", doc("tsig")},
	"cache":        {"enables a frontend cache.", doc("cache")},
	"rewrite":      {"performs internal message rewriting.", doc("rewrite")},
	"header":       {"modifies the header for queries and responses.", doc("header")},
	"dnssec":       {"enables on-the-fly DNSSEC signing of served data.", doc("dnssec")},
	"autopath":     {"allows for server-side search path completion.", doc("autopath")},
	"minimal":      {"minimizes the size of the DNS response message when possible.", doc("minimal")},
	"template":     {"allows for dynamic responses based on the incoming query.", doc("template")},
	"transfer":     {"performs outgoing zone transfers for other plugins.", doc("transfer")},
	"hosts":        {"enables serving zone data from an /etc/hosts style file.", doc("hosts")},
	"route53":      {"enables serving zone data from AWS Route 53.", doc("route53")},
	"azure":        {"enables serving zone data from Microsoft Azure DNS.", doc("azure")},
	"clouddns":     {"enables serving zone data from GCP Cloud DNS.", doc("clouddns")},
	"k8s_external": {"resolves load balancer and external IPs from outside Kubernetes clusters.", doc("k8s_external")},
	"kubernetes":   {"enables reading zone data from a Kubernetes cluster.", doc("kubernetes")},
	"file":         {"enables serving zone data from an RFC 1035-style master file.", doc("file")},
	"auto":         {"serves zone data from RFC 1035-style master files picked up automatically from disk.", doc("auto")},
	"secondary":    {"enables serving a zone retrieved from a primary server.", doc("secondary")},
	"etcd":         {"enables SkyDNS service discovery from etcd.", doc("etcd")},
	"loop":         {"detects simple forwarding loops and halts the server.", doc("loop")},
	"forward":      {"facilitates proxying DNS messages to upstream resolvers.", doc("forward")},
	"grpc":         {"proxies DNS messages to upstream resolvers via gRPC.", doc("grpc")},
	"erratic":      {"a plugin useful for testing client behavior.", doc("erratic")},
	"whoami":       {"returns the resolver's local IP address, port and transport.", doc("whoami")},
	"on":           {"executes shell commands on startup, shutdown and restart events.", ""},
	"sign":         {"adds DNSSEC records to zone files.", doc("sign")},
	"view":         {"defines conditions a DNS request must meet to be routed to the server block.", doc("view")},
}

// Catalog returns the plugin reference metadata keyed by plugin name. Callers
// must not mutate the returned map.
func Catalog() map[string]Meta { return meta }
