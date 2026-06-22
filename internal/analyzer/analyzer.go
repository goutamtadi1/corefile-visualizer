// Package analyzer parses a CoreDNS Corefile into the structured model. It
// walks the caddy tokenizer's flat, ordered token stream so that directive
// order and repeated directives are preserved (unlike caddyfile.Parse, which
// groups directives into an order-losing map).
package analyzer

import (
	"errors"
	"strings"

	"github.com/coredns/caddy/caddyfile"
	"github.com/gtadi/corefile-visualizer/internal/model"
)

// ErrMissingCloseBrace is returned when a server block or nested block is not
// closed before end of input.
var ErrMissingCloseBrace = errors.New("missing closing brace '}'")

// ErrMissingOpenBrace is returned when server-block keys are not followed by '{'.
var ErrMissingOpenBrace = errors.New("missing opening brace '{'")

type token struct {
	text string
	line int
}

type parser struct {
	tokens []token
	pos    int
}

// Analyze parses the Corefile text into a *model.Corefile.
func Analyze(input string) (*model.Corefile, error) {
	d := caddyfile.NewDispenser("Corefile", strings.NewReader(input))
	var ts []token
	for d.Next() {
		ts = append(ts, token{text: d.Val(), line: d.Line()})
	}

	p := &parser{tokens: ts}
	cf := &model.Corefile{}
	for p.pos < len(p.tokens) {
		sb, err := p.parseServerBlock()
		if err != nil {
			return nil, err
		}
		cf.ServerBlocks = append(cf.ServerBlocks, *sb)
	}
	return cf, nil
}

func (p *parser) parseServerBlock() (*model.ServerBlock, error) {
	sb := &model.ServerBlock{Line: p.tokens[p.pos].line}
	for p.pos < len(p.tokens) && p.tokens[p.pos].text != "{" {
		sb.Keys = append(sb.Keys, p.tokens[p.pos].text)
		p.pos++
	}
	if p.pos >= len(p.tokens) {
		return nil, ErrMissingOpenBrace
	}
	p.pos++ // consume "{"

	dirs, err := p.parseDirectives()
	if err != nil {
		return nil, err
	}
	sb.Directives = dirs
	return sb, nil
}

// parseDirectives consumes directives until a matching "}" (which it consumes).
func (p *parser) parseDirectives() ([]model.Directive, error) {
	dirs := []model.Directive{}
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.text == "}" {
			p.pos++
			return dirs, nil
		}

		d := model.Directive{Name: t.text, Line: t.line}
		p.pos++

		for p.pos < len(p.tokens) {
			n := p.tokens[p.pos]
			if n.text == "}" {
				break
			}
			if n.text == "{" {
				p.pos++ // consume "{"
				blk, err := p.parseDirectives()
				if err != nil {
					return nil, err
				}
				d.Block = blk
				break
			}
			if n.line != t.line {
				break // next directive begins on a new line
			}
			d.Args = append(d.Args, n.text)
			p.pos++
		}
		dirs = append(dirs, d)
	}
	return nil, ErrMissingCloseBrace
}
