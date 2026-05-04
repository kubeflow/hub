module github.com/kubeflow/hub

go 1.25.7

require (
	cirello.io/pglock v1.16.1
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/alecthomas/participle/v2 v2.1.4
	github.com/deckarep/golang-set/v2 v2.9.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	github.com/go-sql-driver/mysql v1.10.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/golang/glog v1.2.5
	github.com/jackc/pgx/v5 v5.9.2
	github.com/kubeflow/hub/catalog/pkg/openapi v0.0.0-00010101000000-000000000000
	github.com/kubeflow/hub/pkg/openapi v0.0.0
	github.com/lib/pq v1.12.3
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.42.0
	github.com/testcontainers/testcontainers-go/modules/mysql v0.42.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.42.0
	golang.org/x/sync v0.20.0
	google.golang.org/protobuf v1.36.11
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
	k8s.io/apimachinery v0.35.4
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.2.0 // indirect
	github.com/moby/moby/api v1.54.1 // indirect
	github.com/moby/moby/client v0.4.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/shirou/gopsutil/v4 v4.26.3 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	go.mongodb.org/mongo-driver v1.17.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.64.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/google/uuid v1.6.0
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/exp v0.0.0-20250808145144-a408d31f581a
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/kubeflow/hub/pkg/openapi => ./pkg/openapi

replace github.com/kubeflow/hub/catalog/pkg/openapi => ./catalog/pkg/openapi
