package dsl

import (
	"bufio"
	"regexp"
	"strings"
	"sync"
)

var setOutputPattern = regexp.MustCompile(`::set-output name=([^:]+)::(.*)`)

// OutputParser parses step output protocol (::set-output name=key::value)
// It is safe for concurrent use.
type OutputParser struct {
	mu      sync.RWMutex
	outputs map[string]string
}

// NewOutputParser creates a new output parser
func NewOutputParser() *OutputParser {
	return &OutputParser{
		outputs: make(map[string]string),
	}
}

// ParseLine parses a single line of output
func (p *OutputParser) ParseLine(line string) {
	matches := setOutputPattern.FindStringSubmatch(line)
	if len(matches) == 3 {
		name := strings.TrimSpace(matches[1])
		value := strings.TrimSpace(matches[2])
		p.mu.Lock()
		p.outputs[name] = value
		p.mu.Unlock()
	}
}

// ParseOutput parses complete output and extracts all ::set-output commands
func (p *OutputParser) ParseOutput(output string) map[string]string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		p.ParseLine(scanner.Text())
	}
	p.mu.RLock()
	result := make(map[string]string, len(p.outputs))
	for k, v := range p.outputs {
		result[k] = v
	}
	p.mu.RUnlock()
	return result
}

// GetOutputs returns all parsed outputs
func (p *OutputParser) GetOutputs() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]string, len(p.outputs))
	for k, v := range p.outputs {
		result[k] = v
	}
	return result
}
