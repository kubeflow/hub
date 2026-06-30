package datastore

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

var (
	ErrCreatingDatastore    = errors.New("error creating datastore")
	ErrUnsupportedDatastore = errors.New("unsupported datastore type")
)

type Connector interface {
	Type() string
	Connect(spec *Spec) (RepoSet, error)
	RunMigrations(spec *Spec) error
	// Reconnect creates a fresh RepoSet after the database has been recreated
	// (e.g., by RunMigrations recovering from data loss). It must not re-run
	// migrations; callers are responsible for calling RunMigrations first.
	Reconnect(spec *Spec) (RepoSet, error)
}

var connectorTypes map[string]func(any) (Connector, error)

func Register(t string, fn func(config any) (Connector, error)) {
	if connectorTypes == nil {
		connectorTypes = make(map[string]func(any) (Connector, error), 1)
	}

	if _, exists := connectorTypes[t]; exists {
		panic(fmt.Sprintf("duplicate connector type: %s", t))
	}

	connectorTypes[t] = fn
}

func NewConnector(t string, config any) (Connector, error) {
	if fn, ok := connectorTypes[t]; ok {
		return fn(config)
	}

	return nil, fmt.Errorf("%w: %s. Supported types: %v", ErrUnsupportedDatastore, t, slices.Sorted(maps.Keys(connectorTypes)))
}
