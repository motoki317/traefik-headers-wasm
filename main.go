package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

func main() {
	// No buffer request is required if we are just reading request URI and headers (and not body)
	// handler.Host.EnableFeatures(api.FeatureBufferRequest)

	var config Config
	err := json.Unmarshal(handler.Host.GetConfig(), &config)
	if err != nil {
		handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not load config %v", err))
		os.Exit(1)
	}

	mw, err := New(&config)
	if err != nil {
		handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not load config %v", err))
		os.Exit(1)
	}
	handler.HandleRequestFn = mw.handleRequest
	handler.Host.Log(api.LogLevelDebug, fmt.Sprintf("[traefik-headers-wasm] Loaded plugin with %d manipulation(s)", len(config.Manipulations)))
}

// Config is the plugin configuration.
type Config struct {
	Manipulations []manipulationConfig `json:"manipulations"`
}

type manipulationConfig struct {
	MatchPath          string             `json:"matchPath"`
	MatchRequestHeader *matchHeaderConfig `json:"matchRequestHeader"`

	CustomRequestHeaders  []customHeader `json:"customRequestHeaders"`
	CustomResponseHeaders []customHeader `json:"customResponseHeaders"`
}

type matchHeaderConfig struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (m *manipulationConfig) compile() (*manipulation, error) {
	var man manipulation
	var err error
	man.matcher, err = m.getMatcher()
	if err != nil {
		return nil, err
	}
	for _, h := range m.CustomRequestHeaders {
		hh, err := h.compile()
		if err != nil {
			return nil, err
		}
		man.customRequestHeaders = append(man.customRequestHeaders, hh)
	}
	for _, h := range m.CustomResponseHeaders {
		hh, err := h.compile()
		if err != nil {
			return nil, err
		}
		man.customResponseHeaders = append(man.customResponseHeaders, hh)
	}
	return &man, nil
}

func (m *manipulationConfig) getMatcher() (matcher, error) {
	// Check that only one of MatchPath or MatchRequestHeader is set
	if m.MatchPath != "" && m.MatchRequestHeader != nil {
		return nil, fmt.Errorf("matchPath and matchRequestHeader are mutually exclusive")
	}
	if m.MatchPath == "" && m.MatchRequestHeader == nil {
		return nil, fmt.Errorf("either matchPath or matchRequestHeader must be set")
	}

	if m.MatchPath != "" {
		re, err := regexp.Compile(m.MatchPath)
		if err != nil {
			return nil, err
		}
		return &fromPathMatcher{re: re}, nil
	}

	// Handle header matching
	if m.MatchRequestHeader.Name == "" {
		return nil, fmt.Errorf("matchRequestHeader.name is required")
	}
	if m.MatchRequestHeader.Value == "" {
		return nil, fmt.Errorf("matchRequestHeader.value is required")
	}
	re, err := regexp.Compile(m.MatchRequestHeader.Value)
	if err != nil {
		return nil, err
	}
	return &fromHeaderMatcher{headerName: m.MatchRequestHeader.Name, re: re}, nil
}

type fromPathMatcher struct {
	re *regexp.Regexp
}

func (f *fromPathMatcher) match(req api.Request) bool {
	return f.re.MatchString(req.GetURI())
}

func (f *fromPathMatcher) replace(req api.Request, tmpl string) string {
	return f.re.ReplaceAllString(req.GetURI(), tmpl)
}

type fromHeaderMatcher struct {
	headerName string
	re         *regexp.Regexp
}

func (f *fromHeaderMatcher) match(req api.Request) bool {
	headerValue, ok := req.Headers().Get(f.headerName)
	if !ok {
		return false
	}
	return f.re.MatchString(headerValue)
}

func (f *fromHeaderMatcher) replace(req api.Request, tmpl string) string {
	headerValue, ok := req.Headers().Get(f.headerName)
	if !ok {
		return ""
	}
	return f.re.ReplaceAllString(headerValue, tmpl)
}

type customHeader struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Replace bool   `json:"replace"`
}

func (h *customHeader) compile() (*headerManipulation, error) {
	if h.Name == "" {
		return nil, fmt.Errorf("header name is required")
	}
	return &headerManipulation{
		name:    h.Name,
		tmpl:    h.Value,
		replace: h.Replace,
	}, nil
}

// matcher matches request and replaces strings.
type matcher interface {
	match(req api.Request) bool
	replace(req api.Request, tmpl string) string
}

type manipulation struct {
	matcher matcher

	customRequestHeaders  []*headerManipulation
	customResponseHeaders []*headerManipulation
}

type headerManipulation struct {
	name    string
	tmpl    string
	replace bool
}

// Plugin is a plugin instance.
type Plugin struct {
	manipulations []*manipulation
}

// New creates a new plugin instance.
func New(c *Config) (*Plugin, error) {
	var p Plugin

	for _, m := range c.Manipulations {
		man, err := m.compile()
		if err != nil {
			return nil, err
		}
		p.manipulations = append(p.manipulations, man)
	}

	return &p, nil
}

func (p *Plugin) handleRequest(req api.Request, res api.Response) (next bool, reqCtx uint32) {
	for _, m := range p.manipulations {
		match := m.matcher.match(req)
		if !match {
			continue
		}

		for _, h := range m.customRequestHeaders {
			result := m.matcher.replace(req, h.tmpl)
			if h.replace {
				req.Headers().Set(h.name, result)
			} else {
				req.Headers().Add(h.name, result)
			}
		}
		for _, h := range m.customResponseHeaders {
			result := m.matcher.replace(req, h.tmpl)
			if h.replace {
				res.Headers().Set(h.name, result)
			} else {
				res.Headers().Add(h.name, result)
			}
		}
	}

	return true, 0
}
