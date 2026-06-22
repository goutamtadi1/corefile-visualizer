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
