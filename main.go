package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

func main() {
	handler.Host.EnableFeatures(api.FeatureBufferRequest)

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
	MatchPath string `json:"matchPath"`

	CustomRequestHeaders  []customHeader `json:"customRequestHeaders"`
	CustomResponseHeaders []customHeader `json:"customResponseHeaders"`
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
	re, err := regexp.Compile(m.MatchPath)
	if err != nil {
		return nil, err
	}
	return func(req api.Request) (bool, map[string]string) {
		return matchSubexp(re, req.GetURI())
	}, nil
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
	tmpl, err := template.New("header").Parse(h.Value)
	if err != nil {
		return nil, err
	}
	return &headerManipulation{
		name:    h.Name,
		tmpl:    tmpl,
		replace: h.Replace,
	}, nil
}

// matcher matches request and replaces strings.
type matcher = func(req api.Request) (bool, map[string]string)

type manipulation struct {
	matcher matcher

	customRequestHeaders  []*headerManipulation
	customResponseHeaders []*headerManipulation
}

type headerManipulation struct {
	name    string
	tmpl    *template.Template
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
		match, subexp := m.matcher(req)
		handler.Host.Log(api.LogLevelDebug, fmt.Sprintf("Match: %v, subexp: %v", match, subexp))
		if !match {
			continue
		}

		for _, h := range m.customRequestHeaders {
			result, err := executeTmpl(h.tmpl, subexp)
			if err != nil {
				handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not execute template: %v", err))
				return false, 0
			}
			if h.replace {
				req.Headers().Set(h.name, result)
			} else {
				req.Headers().Add(h.name, result)
			}
		}
		for _, h := range m.customResponseHeaders {
			result, err := executeTmpl(h.tmpl, subexp)
			if err != nil {
				handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not execute template: %v", err))
				return false, 0
			}
			if h.replace {
				res.Headers().Set(h.name, result)
			} else {
				res.Headers().Add(h.name, result)
			}
		}
	}

	return true, 0
}
