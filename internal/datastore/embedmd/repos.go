package embedmd

import (
	"context"
	"fmt"
	"maps"
	"reflect"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/internal/db/service"
	"github.com/kubeflow/hub/internal/platform/datastore"
	"gorm.io/gorm"
)

type ErrMissingType string

func (t ErrMissingType) Error() string {
	if string(t) == "" {
		return "no types available"
	}
	return fmt.Sprintf("required type '%s' not found in database. Please ensure all migrations have been applied", string(t))
}

var _ datastore.RepoSet = (*repoSetImpl)(nil)

type repoSetImpl struct {
	db        *gorm.DB
	spec      *datastore.Spec
	nameIDMap map[string]int32
	repos     map[reflect.Type]any
}

func newRepoSet(db *gorm.DB, spec *datastore.Spec) (datastore.RepoSet, error) {
	return newRepoSetContext(db.Statement.Context, db, spec)
}

func newRepoSetContext(ctx context.Context, db *gorm.DB, spec *datastore.Spec) (datastore.RepoSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	typeRepository := service.NewTypeRepository(db.WithContext(ctx))

	glog.Infof("Getting types...")

	types, err := typeRepository.GetAll()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMissingType(""), err)
	}

	nameIDMap := make(map[string]int32, len(types))
	for _, t := range types {
		nameIDMap[*t.GetAttributes().Name] = int32(*t.GetID())
	}

	glog.Infof("Types retrieved")

	// Add debug logging to see what types are actually available
	glog.V(2).Infof("DEBUG: Available types:")
	for typeName, typeID := range nameIDMap {
		glog.V(2).Infof("  %s = %d", typeName, typeID)
	}

	// Validate that all required types are registered
	requiredTypes := spec.AllNames()
	for _, requiredType := range requiredTypes {
		if _, exists := nameIDMap[requiredType]; !exists {
			return nil, ErrMissingType(requiredType)
		}
	}

	glog.Infof("All required types validated successfully")

	rs := &repoSetImpl{
		db:        db,
		spec:      spec,
		nameIDMap: nameIDMap,
		repos:     make(map[reflect.Type]any, len(requiredTypes)+1),
	}

	artifactTypes := makeTypeMap[datastore.ArtifactTypeMap](spec.ArtifactTypes, nameIDMap)
	contextTypes := makeTypeMap[datastore.ContextTypeMap](spec.ContextTypes, nameIDMap)
	executionTypes := makeTypeMap[datastore.ExecutionTypeMap](spec.ExecutionTypes, nameIDMap)

	args := map[reflect.Type]any{
		reflect.TypeFor[*gorm.DB]():                   db,
		reflect.TypeFor[datastore.ArtifactTypeMap]():  artifactTypes,
		reflect.TypeFor[datastore.ContextTypeMap]():   contextTypes,
		reflect.TypeFor[datastore.ExecutionTypeMap](): executionTypes,
	}

	for i, fn := range spec.Others {
		repo, err := rs.call(fn, args)
		if err != nil {
			return nil, fmt.Errorf("embedmd: other %d: %w", i, err)
		}
		rs.putWithReturnType(fn, repo)
	}

	for name, specType := range spec.ArtifactTypes {
		args[reflect.TypeFor[int32]()] = nameIDMap[name]

		repo, err := rs.call(specType.InitFn, args)
		if err != nil {
			return nil, fmt.Errorf("embedmd: %s: %w", name, err)
		}
		rs.putWithReturnType(specType.InitFn, repo)
	}

	for name, specType := range spec.ContextTypes {
		args[reflect.TypeFor[int32]()] = nameIDMap[name]

		repo, err := rs.call(specType.InitFn, args)
		if err != nil {
			return nil, fmt.Errorf("embedmd: %s: %w", name, err)
		}
		rs.putWithReturnType(specType.InitFn, repo)
	}

	for name, specType := range spec.ExecutionTypes {
		args[reflect.TypeFor[int32]()] = nameIDMap[name]
		repo, err := rs.call(specType.InitFn, args)
		if err != nil {
			return nil, fmt.Errorf("embedmd: %s: %w", name, err)
		}
		rs.putWithReturnType(specType.InitFn, repo)
	}

	return rs, nil
}

// call invokes the function pointed to by fn. It matches fn's arguments to the
// types in args. fn must return at least one argument, and may optionally
// return an error.
func (rs *repoSetImpl) call(fn any, args map[reflect.Type]any) (any, error) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("initializer is not a function (got type %T)", fn)
	}

	switch t.NumOut() {
	case 0:
		return nil, fmt.Errorf("initializer has no return value")
	case 1, 2:
		// OK
	default:
		return nil, fmt.Errorf("unknown initializer type, more than 2 return values")
	}

	fnArgs := make([]reflect.Value, t.NumIn())
	for i := range t.NumIn() {
		v, ok := args[t.In(i)]
		if !ok {
			return nil, fmt.Errorf("no initializer argument for type %v", t.In(i))
		}
		fnArgs[i] = reflect.ValueOf(v)
	}

	out := reflect.ValueOf(fn).Call(fnArgs)

	var err error
	if len(out) > 1 {
		ierr := out[1].Interface()
		if ierr != nil {
			var ok bool
			err, ok = ierr.(error)
			if !ok {
				return nil, fmt.Errorf("unknown return value, expected error, got %T", err)
			}
		}
	}

	return out[0].Interface(), err
}

// put adds one repository to the set, keyed by its concrete type.
func (rs *repoSetImpl) put(repo any) {
	rs.repos[reflect.TypeOf(repo)] = repo
}

// putWithReturnType stores the repo by both its concrete type and the declared
// return type of the init function. This ensures that GetRepo[InterfaceType]
// finds the correct repo via exact match, avoiding the non-deterministic
// Implements fallback when multiple generic repos share the same underlying
// structure (e.g., multiple GenericRepository[T, ...] for different context types).
func (rs *repoSetImpl) putWithReturnType(fn any, repo any) {
	rs.put(repo)
	fnType := reflect.TypeOf(fn)
	if fnType.NumOut() > 0 {
		rs.repos[fnType.Out(0)] = repo
	}
}

func (rs *repoSetImpl) Repository(t reflect.Type) (any, error) {
	// First try an exact match for the requested type.
	repo, ok := rs.repos[t]
	if ok {
		return repo, nil
	}

	// If the attempt above failed and the requested type is an interface,
	// use the first repo that implements it.
	if t.Kind() == reflect.Interface {
		for repoType, repo := range rs.repos {
			if repoType.Implements(t) {
				return repo, nil
			}
		}
	}

	return nil, fmt.Errorf("unknown repository type: %s", t.Name())
}

func (rs *repoSetImpl) TypeMap() map[string]int32 {
	clone := make(map[string]int32, len(rs.nameIDMap))
	maps.Copy(clone, rs.nameIDMap)
	return clone
}

func makeTypeMap[T ~map[string]int32](specMap map[string]*datastore.SpecType, nameIDMap map[string]int32) T {
	returnMap := make(T, len(specMap))
	for k := range specMap {
		returnMap[k] = nameIDMap[k]
	}
	return returnMap
}
