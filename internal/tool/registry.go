package tool

import (
	"io"
	"sort"
)

// Registry holds all available tools.
type Registry struct {
	tools map[string]*Tool
}

// NewRegistry creates a registry with all available tools.
func NewRegistry(progress io.Writer) *Registry {
	return &Registry{
		tools: map[string]*Tool{
			"kubectl": NewKubectl(progress),
			"kind":    NewKind(progress),
		},
	}
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) *Tool {
	return r.tools[name]
}

// All returns all registered tool names sorted alphabetically.
func (r *Registry) All() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// AllTools returns all registered tools sorted alphabetically by name.
func (r *Registry) AllTools() []*Tool {
	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}
