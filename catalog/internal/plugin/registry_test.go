package plugin

import (
	"context"
	"flag"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type mockPlugin struct {
	name        string
	version     string
	description string
	healthy     bool
}

func (p *mockPlugin) Name() string                               { return p.name }
func (p *mockPlugin) Version() string                            { return p.version }
func (p *mockPlugin) Description() string                        { return p.description }
func (p *mockPlugin) Init(_ context.Context, _ Config) error     { return nil }
func (p *mockPlugin) Start(_ context.Context) error              { return nil }
func (p *mockPlugin) Stop(_ context.Context) error               { return nil }
func (p *mockPlugin) Healthy() bool                              { return p.healthy }
func (p *mockPlugin) RegisterRoutes(_ chi.Router) error          { return nil }
func (p *mockPlugin) Migrations() []Migration                    { return nil }

func TestRegister(t *testing.T) {
	Reset()
	defer Reset()

	plugin := &mockPlugin{
		name:        "test-plugin",
		version:     "v1",
		description: "Test plugin",
		healthy:     true,
	}

	Register(plugin)

	names := Names()
	assert.Equal(t, 1, len(names))
	assert.Equal(t, "test-plugin", names[0])

	p, ok := Get("test-plugin")
	assert.True(t, ok)
	assert.Equal(t, "test-plugin", p.Name())
	assert.Equal(t, "v1", p.Version())

	_, ok = Get("non-existent")
	assert.False(t, ok)
}

func TestRegisterDuplicate(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "duplicate"})

	assert.Panics(t, func() {
		Register(&mockPlugin{name: "duplicate"})
	})
}

func TestAll(t *testing.T) {
	Reset()
	defer Reset()

	Register(&mockPlugin{name: "plugin-a"})
	Register(&mockPlugin{name: "plugin-b"})
	Register(&mockPlugin{name: "plugin-c"})

	all := All()
	assert.Equal(t, 3, len(all))
	assert.Equal(t, "plugin-a", all[0].Name())
	assert.Equal(t, "plugin-b", all[1].Name())
	assert.Equal(t, "plugin-c", all[2].Name())
}

type flagPlugin struct {
	mockPlugin
	registered bool
}

func (p *flagPlugin) RegisterFlags(fs *flag.FlagSet) {
	p.registered = true
	fs.Bool("test-flag", false, "test")
}

func TestRegisterAllFlags(t *testing.T) {
	Reset()
	defer Reset()

	plain := &mockPlugin{name: "plain"}
	withFlags := &flagPlugin{mockPlugin: mockPlugin{name: "with-flags"}}

	Register(plain)
	Register(withFlags)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	RegisterAllFlags(fs)

	assert.True(t, withFlags.registered)
	assert.NotNil(t, fs.Lookup("test-flag"))
}

func TestCount(t *testing.T) {
	Reset()
	defer Reset()

	assert.Equal(t, 0, Count())

	Register(&mockPlugin{name: "p1"})
	assert.Equal(t, 1, Count())

	Register(&mockPlugin{name: "p2"})
	assert.Equal(t, 2, Count())

	Reset()
	assert.Equal(t, 0, Count())
}
