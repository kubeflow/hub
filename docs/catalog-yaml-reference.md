# Catalog YAML Reference

This document describes the YAML configuration formats for the Kubeflow Hub catalog component. It covers two distinct layers:

1. **Sources configuration** — how to register catalog sources (YAML files, Hugging Face Hub, etc.)
2. **Catalog data files** — the YAML format for defining model and MCP server entries

---

## Table of Contents

- [Sources Configuration](#sources-configuration)
  - [Top-Level Structure](#top-level-structure)
  - [Source Types](#source-types)
  - [YAML Source Type](#yaml-source-type)
  - [Hugging Face Hub Source Type](#hugging-face-hub-source-type)
  - [Named Queries](#named-queries)
  - [Labels](#labels)
- [Model Catalog Data Files](#model-catalog-data-files)
  - [Model Fields](#model-fields)
  - [Validated Tasks and Serving Configuration](#validated-tasks-and-serving-configuration)
  - [Model Artifacts](#model-artifacts)
  - [Metrics Artifacts](#metrics-artifacts)
- [MCP Server Catalog Data Files](#mcp-server-catalog-data-files)
  - [MCP Server Fields](#mcp-server-fields)
  - [Tools](#tools)
  - [Security Indicators](#security-indicators)
  - [Artifacts (Local Servers)](#artifacts-local-servers)
  - [Endpoints (Remote Servers)](#endpoints-remote-servers)
  - [Runtime Metadata](#runtime-metadata)
- [Custom Properties](#custom-properties)
- [Minimal Examples](#minimal-examples)
- [Complete Examples](#complete-examples)

---

## Sources Configuration

The sources configuration file tells the catalog server where to find model and MCP server data. It is loaded at startup and mounted as a ConfigMap in Kubernetes.

### Top-Level Structure

```yaml
# Model catalog sources
model_catalogs:
  - name: "My Models"
    id: my_models
    type: yaml                # or "hf" for Hugging Face Hub
    enabled: true
    properties:
      yamlCatalogPath: my-models.yaml
    labels:
      - My Organization
    includedModels:            # Optional glob patterns to include
      - "my-org/*"
    excludedModels:            # Optional glob patterns to exclude
      - "*-draft"

# MCP server catalog sources
mcp_catalogs:
  - name: "My MCP Servers"
    id: my_mcp_servers
    type: yaml
    enabled: true
    properties:
      yamlCatalogPath: my-mcp-servers.yaml
    labels:
      - My Organization
    includedServers:           # Optional glob patterns to include
      - "prod-*"
    excludedServers:           # Optional glob patterns to exclude
      - "*-deprecated"

# Named queries (reusable filter presets for both models and MCP servers)
namedQueries:
  production_ready:
    verifiedSource:
      operator: "="
      value: true

# Labels (categorization metadata for sources)
labels:
  - name: my-label
    displayName: My Label
    assetType: models          # "models" or "mcp_servers"
```

> **Note:** The legacy `catalogs` key is deprecated. Use `model_catalogs` instead. If both are present, `model_catalogs` takes precedence for entries with the same ID.

### Source Types

The catalog supports multiple source types via the `type` field:

| Type | Description |
|------|-------------|
| `yaml` | Models or MCP servers defined in a local YAML data file |
| `hf` | Models fetched from the Hugging Face Hub API |

### YAML Source Type

The `yaml` type loads model or MCP server definitions from a local YAML file bundled in the ConfigMap.

| Source Field | Type | Required | Description |
|--------------|------|----------|-------------|
| `name` | string | **Yes** | Human-readable name for this source |
| `id` | string | **Yes** | Unique identifier (no duplicates across model and MCP sources) |
| `type` | string | **Yes** | Must be `yaml` |
| `enabled` | boolean | No | Whether this source is active (default: `true`) |
| `properties.yamlCatalogPath` | string | **Yes** | Path to the YAML data file |
| `labels` | string[] | No | Labels for filtering and categorization |
| `includedModels` | string[] | No | Glob patterns to include (model sources only) |
| `excludedModels` | string[] | No | Glob patterns to exclude (model sources only) |
| `includedServers` | string[] | No | Glob patterns to include (MCP sources only) |
| `excludedServers` | string[] | No | Glob patterns to exclude (MCP sources only) |

Glob pattern rules:
- Only the `*` wildcard is supported (matches zero or more characters)
- Patterns are case-insensitive
- Exclusions take precedence over inclusions
- A pattern cannot appear in both included and excluded lists

### Hugging Face Hub Source Type

The `hf` type fetches model metadata directly from the Hugging Face Hub API. This source type is only available for `model_catalogs`.

```yaml
model_catalogs:
  # Fetch specific models from Hugging Face
  - name: Hugging Face Hub - Specific Models
    id: hf-specific
    type: hf
    enabled: true
    properties:
      apiKeyEnvVar: "HF_API_KEY"    # Env var name holding the API key
    includedModels:
      - "meta-llama/Llama-3.2-1B"
      - "microsoft/phi-2"
    excludedModels:
      - "some-org/*"

  # Restrict to a single Hugging Face organization
  - name: Meta LLaMA Models
    id: hf-meta-llama
    type: hf
    enabled: true
    properties:
      apiKeyEnvVar: "HF_API_KEY"
      allowedOrganization: "meta-llama"   # Auto-prefixes patterns
    includedModels:
      - "*"              # Expands to: meta-llama/*
      - "Llama-3*"       # Expands to: meta-llama/Llama-3*
    excludedModels:
      - "*-GGUF"         # Expands to: meta-llama/*-GGUF
```

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `apiKeyEnvVar` | string | No | Name of the environment variable containing the HF API key (default: `HF_API_KEY`) |
| `allowedOrganization` | string | No | Restricts to a single HF organization; auto-prefixes all patterns with `org/` |

The API key value itself should be stored in a Kubernetes Secret and exposed as an environment variable in the pod configuration.

### Named Queries

Named queries define reusable server-side filter presets that clients can reference by name via the `namedQuery` API parameter. They apply to **both models and MCP servers**.

```yaml
namedQueries:
  # Filter servers/models where all tools are read-only
  read_only_servers:
    readOnlyTools:
      operator: "="
      value: true

  # Combine multiple filter conditions
  fully_audited:
    verifiedSource:
      operator: "="
      value: true
    secureEndpoint:
      operator: "="
      value: true
    sast:
      operator: "="
      value: true

  # Performance threshold filter
  low_latency:
    ttft_p90:
      operator: "<"
      value: 100
```

Each named query is a map of field names to filter conditions. Supported operators: `=`, `!=`, `>`, `<`, `>=`, `<=`, `LIKE`, `ILIKE`, `IN`, `NOT IN`. The `IN` and `NOT IN` operators require array values.

Multiple config files can contribute queries; later files override individual fields within a query of the same name.

### Labels

Top-level `labels` define categorization metadata for sources:

```yaml
labels:
  - name: my-label
    displayName: My Display Name
    assetType: models           # Scope label to "models" or "mcp_servers"
```

---

## Model Catalog Data Files

A model data file has a `source` name and a `models` array:

```yaml
source: My Organization
models:
  - name: org/my-model-7b
    # ... fields described below
```

### Model Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | **Yes** | Unique model name, typically `org/model-name` |
| `provider` | string | No | Organization or entity providing the model |
| `description` | string | No | Human-readable summary (supports multi-line with `\|-`) |
| `readme` | string | No | Full documentation in Markdown |
| `language` | string[] | No | Supported languages as ISO 639 codes (e.g., `["en", "es"]`) or `["multilingual"]` |
| `license` | string | No | SPDX license identifier (e.g., `apache-2.0`, `mit`, `cc-by-sa-4.0`) |
| `licenseLink` | string (URI) | No | URL to the full license text |
| `libraryName` | string | No | ML library name (e.g., `transformers`, `sentence-transformers`) |
| `tasks` | string[] | No | Task identifiers the model is designed for |
| `validatedTasks` | string[] | No | Tasks validated by the catalog provider (see [Validated Tasks and Serving Configuration](#validated-tasks-and-serving-configuration)) |
| `servingConfig` | object | No | Serving and deployment configuration (see [Validated Tasks and Serving Configuration](#validated-tasks-and-serving-configuration)) |
| `maturity` | string | No | Maturity level (e.g., `Production`) |
| `logo` | string (URI) | No | Logo image (data URI or URL) |
| `externalId` | string | No | External identifier from your system (must be unique) |
| `customProperties` | object | No | Key-value metadata (see [Custom Properties](#custom-properties)) |
| `artifacts` | array | No | Model artifacts and metrics (see below) |
| `createTimeSinceEpoch` | string | No | Creation timestamp in milliseconds since Unix epoch |
| `lastUpdateTimeSinceEpoch` | string | No | Last update timestamp in milliseconds since epoch |

**Common task identifiers:** `text-generation`, `question-answering`, `summarization`, `translation`, `text-classification`, `sentiment-analysis`, `feature-extraction`, `sentence-similarity`, `text-embedding`, `image-to-text`, `visual-question-answering`, `conversational`, `code-generation`, `fill-mask`, `table-question-answering`, `tool-calling`

### Validated Tasks and Serving Configuration

The catalog uses a three-tier capability model to distinguish between what a model claims it can do, what has been independently verified, and what runtime configuration is needed to enable a capability.

#### Three-Tier Capability Model

| Layer | Field | Purpose | Filterable |
|-------|-------|---------|-----------|
| Declaration | `tasks` | Model declares a capability | Yes |
| Validation | `validatedTasks` | Capability validated end-to-end by the catalog provider | Yes |
| Configuration | `servingConfig` | Runtime arguments to enable the capability | No |

**Invariants enforced by the catalog pipeline:**

- Every entry in `validatedTasks` must also appear in `tasks`. A model cannot claim validation for a task it does not declare.
- A model with `servingConfig.toolCalling` should have `"tool-calling"` in both `tasks` and `validatedTasks`.
- A model may list `"tool-calling"` in `tasks` but not in `validatedTasks` — this means the capability is declared but not yet validated.

> **Note:** `tasks` and `validatedTasks` are filterable via the catalog API. `servingConfig` is not filterable — it provides deployment-time configuration only.

#### validatedTasks

An array of task identifiers (from the same vocabulary as `tasks`) that have been validated end-to-end by the catalog provider. When absent or empty, no tasks have been validated for the model.

```yaml
validatedTasks:
  - text-generation
  - question-answering
```

#### servingConfig

Serving and deployment configuration for the model. Each property represents a distinct configuration concern with its own typed schema. All properties are optional.

Currently, `servingConfig` supports one configuration type:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `toolCalling` | object | No | Tool-calling arguments for this model (see below) |

#### toolCalling (ToolCallingConfig)

Tool-calling arguments for this model. All fields are optional.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `toolCallParser` | string | No | The tool-call parser identifier for the serving runtime |
| `chatTemplate` | string | No | Path to the chat template file packaged with the model |
| `enableAutoToolChoice` | boolean | No | Whether to enable automatic tool choice in the serving runtime (default: `true`) |
| `requiredArgs` | string[] | No | Additional CLI arguments required for tool calling beyond the parser, template, and auto-tool-choice flags. Each entry is a complete argument (e.g., `"--config_format granite"`) |

```yaml
servingConfig:
  toolCalling:
    toolCallParser: granite
    chatTemplate: opt/app-root/template/tool_chat_template_granite.jinja
    enableAutoToolChoice: true
    requiredArgs:
      - "--config_format granite"
```

> **Note:** `chatTemplate` is a path to a file packaged alongside the model, not inline template content.

### Model Artifacts

Models support two artifact types within the `artifacts` array: **model artifacts** (the model binary/container) and **metrics artifacts** (benchmark results).

#### Model Artifact (default)

```yaml
artifacts:
  - uri: oci://registry.example.com/org/model-name:v1.0    # Required
    createTimeSinceEpoch: "1736899200000"
    lastUpdateTimeSinceEpoch: "1736899200000"
    customProperties:
      architecture:
        metadataType: MetadataStringValue
        string_value: '["amd64", "arm64"]'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string (URI) | **Yes** | Location where the model can be retrieved (OCI registry, Hugging Face URL, etc.) |
| `artifactType` | string | No | Defaults to `model-artifact` when omitted |
| `name` | string | No | Optional artifact name |
| `description` | string | No | Optional description |
| `externalId` | string | No | Optional external ID |
| `customProperties` | object | No | Custom metadata (e.g., architecture, validation status) |
| `createTimeSinceEpoch` | string | No | Creation timestamp |
| `lastUpdateTimeSinceEpoch` | string | No | Last update timestamp |

### Metrics Artifacts

Metrics artifacts capture benchmark and performance data. They are distinguished by `artifactType: metrics-artifact`.

#### Performance Metrics

```yaml
artifacts:
  # ... model artifact first, then metrics:
  - artifactType: metrics-artifact
    metricsType: performance-metrics
    createTimeSinceEpoch: "1736899200000"
    lastUpdateTimeSinceEpoch: "1736899200000"
    customProperties:
      config_id:
        string_value: "model-h100-2gpu-chatbot"
        metadataType: MetadataStringValue
      scenario_id:
        string_value: "perf-scenario-1"
        metadataType: MetadataStringValue
      use_case:
        string_value: "chatbot"
        metadataType: MetadataStringValue
      hardware_type:
        string_value: "H100"
        metadataType: MetadataStringValue
      hardware_count:
        int_value: "2"
        metadataType: MetadataIntValue
      ttft_mean:              # Time to first token (ms)
        double_value: 85.2
        metadataType: MetadataDoubleValue
      e2e_mean:               # End-to-end latency (ms)
        double_value: 6850.3
        metadataType: MetadataDoubleValue
      tps_mean:               # Tokens per second
        double_value: 1105.4
        metadataType: MetadataDoubleValue
      itl_mean:               # Inter-token latency (ms)
        double_value: 26.8
        metadataType: MetadataDoubleValue
      requests_per_second:
        double_value: 4.0
        metadataType: MetadataDoubleValue
      framework_type:
        string_value: "vllm"
        metadataType: MetadataStringValue
```

Common performance metric keys: `ttft_mean/p90/p95/p99`, `e2e_mean/p90/p95/p99`, `tps_mean/p90/p95/p99`, `itl_mean/p90/p95/p99`, `requests_per_second`, `mean_input_tokens`, `mean_output_tokens`, `framework_type`, `framework_version`, `deployment_type`.

#### Accuracy Metrics

```yaml
  - artifactType: metrics-artifact
    metricsType: accuracy-metrics
    createTimeSinceEpoch: "1736899200000"
    lastUpdateTimeSinceEpoch: "1736899200000"
    customProperties:
      benchmark:
        string_value: "mmlu"
        metadataType: MetadataStringValue
      score:
        double_value: 75.8
        metadataType: MetadataDoubleValue
      score_metric:
        string_value: "accuracy_percent"
        metadataType: MetadataStringValue
      hardware_type:
        string_value: "H100"
        metadataType: MetadataStringValue
      hardware_count:
        int_value: "2"
        metadataType: MetadataIntValue
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `artifactType` | string | **Yes** | Must be `metrics-artifact` |
| `metricsType` | string | **Yes** | Either `performance-metrics` or `accuracy-metrics` |
| `customProperties` | object | No | Metric key-value pairs (all metrics go here) |

---

## MCP Server Catalog Data Files

An MCP server data file has a `source` name and an `mcp_servers` array:

```yaml
source: My Organization
mcp_servers:
  - name: my-mcp-server
    # ... fields described below
```

### MCP Server Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | **Yes** | Unique server name |
| `provider` | string | No | Organization providing the server |
| `description` | string | No | Human-readable summary |
| `readme` | string | No | Full Markdown documentation |
| `version` | string | No | Semantic version (e.g., `"1.0.0"`, `"latest"`) |
| `license` | string | No | SPDX license identifier |
| `license_link` | string (URI) | No | URL to license text |
| `logo` | string (URI) | No | Logo image (data URI or URL) |
| `tags` | string[] | No | Categorization tags (e.g., `["monitoring", "kubernetes"]`) |
| `transports` | string[] | No | Supported protocols: `stdio`, `http`, `sse` |
| `deploymentMode` | string | No | `local` (default, deployed as a container) or `remote` (hosted externally) |
| `documentationUrl` | string (URI) | No | URL to external documentation |
| `repositoryUrl` | string (URI) | No | URL to source repository |
| `sourceCode` | string | No | Source code location (e.g., `org/repo-name`) |
| `publishedDate` | string | No | Publication date (e.g., `"2025-03-01"`) |
| `externalId` | string | No | External identifier |
| `tools` | array | No | Tools the server exposes (see [Tools](#tools)) |
| `artifacts` | array | No | Container image references for local servers |
| `endpoints` | object | No | Network endpoints for remote servers |
| `securityIndicators` | object | No | Security posture flags (see [Security Indicators](#security-indicators)) |
| `runtimeMetadata` | object | No | Deployment configuration (see [Runtime Metadata](#runtime-metadata)) |
| `customProperties` | object | No | Key-value metadata |
| `createTimeSinceEpoch` | string | No | Creation timestamp (ms since epoch) |
| `lastUpdateTimeSinceEpoch` | string | No | Last update timestamp (ms since epoch) |

### Tools

Each tool describes a capability the MCP server exposes:

```yaml
tools:
  - name: list_pods                  # Required
    description: List pods           # Tool purpose
    accessType: read_only            # read_only | read_write | execute
    parameters:
      - name: namespace             # Required
        type: string                # Required (string, number, object, etc.)
        description: K8s namespace  # Parameter description
        required: true              # Required (boolean)
```

| Tool Field | Type | Required | Description |
|------------|------|----------|-------------|
| `name` | string | **Yes** | Tool identifier |
| `description` | string | No | What the tool does |
| `accessType` | string | **Yes** | `read_only`, `read_write`, or `execute` |
| `parameters` | array | No | Input parameters (use `[]` for none) |

| Parameter Field | Type | Required | Description |
|-----------------|------|----------|-------------|
| `name` | string | **Yes** | Parameter name |
| `type` | string | **Yes** | Data type (`string`, `number`, `object`, etc.) |
| `description` | string | No | What the parameter controls |
| `required` | boolean | **Yes** | Whether the parameter is mandatory |

### Security Indicators

```yaml
securityIndicators:
  verifiedSource: true      # Source code has been verified
  secureEndpoint: true      # Server exposes secure (TLS) endpoints
  sast: true                # Static analysis security testing performed
  readOnlyTools: true       # All tools are read-only (no write operations)
```

All four fields are boolean and optional.

### Artifacts (Local Servers)

For locally-deployed servers, `artifacts` points to the container image:

```yaml
artifacts:
  - uri: oci://registry.example.com/org/mcp-server:1.0.0   # Required
    createTimeSinceEpoch: "1740787200000"
    lastUpdateTimeSinceEpoch: "1740787200000"
```

### Endpoints (Remote Servers)

For remotely-hosted servers (`deploymentMode: remote`), use `endpoints` instead of `artifacts`:

```yaml
deploymentMode: remote
endpoints:
  http: https://api.example.com/mcp/           # HTTP endpoint
  sse: https://api.example.com/mcp/stream      # Server-Sent Events endpoint
  websocket: wss://api.example.com/mcp/ws      # WebSocket endpoint
```

At least one endpoint should be specified. Remote servers do **not** need an `artifacts` section.

### Runtime Metadata

Runtime metadata provides deployment-time information for local servers:

```yaml
runtimeMetadata:
  defaultPort: 8080                    # Port the server listens on
  mcpPath: /mcp                        # HTTP path for MCP requests (default: /mcp)
  defaultArgs:                         # Default command-line arguments
    - --read-only
    - --port=8080

  # Kubernetes prerequisites
  prerequisites:
    serviceAccount:
      required: true
      hint: "Needs 'view' ClusterRole"
      suggestedName: mcp-viewer

    secrets:
      - name: my-credentials
        description: "kubectl create secret generic my-credentials ..."
        keys:
          - key: api-token
            description: API authentication token
            envVarName: API_TOKEN         # Inject as environment variable
            required: true
        mountAsFile: false                # false = env vars, true = file mount

    configMaps:
      - name: server-config
        description: Server configuration
        mountAsFile: true
        mountPath: /etc/config
        keys:
          - key: config.toml
            description: Configuration file
            defaultContent: |
              log_level = 5
              port = "8080"
            required: false

    environmentVariables:
      - name: API_URL
        description: API endpoint URL
        required: true
        type: string

  # Resource recommendations
  recommendedResources:
    minimal:
      cpu: "100m"
      memory: "128Mi"
    recommended:
      cpu: "500m"
      memory: "512Mi"
    high:
      cpu: "2000m"
      memory: "2Gi"

  # Health check endpoints
  healthEndpoints:
    liveness: /healthz
    readiness: /ready

  # Capability flags
  capabilities:
    requiresNetwork: true
    requiresFileSystem: false
    requiresGPU: false
```

---

## Custom Properties

Custom properties use a typed metadata format with three value types:

```yaml
customProperties:
  # String value
  model_type:
    metadataType: MetadataStringValue
    string_value: "generative"

  # Integer value
  hardware_count:
    metadataType: MetadataIntValue
    int_value: "2"                    # Note: specified as a string

  # Double/float value
  accuracy_score:
    metadataType: MetadataDoubleValue
    double_value: 92.4
```

Custom properties function as **tags** when the `string_value` is empty:

```yaml
customProperties:
  kubernetes:
    metadataType: MetadataStringValue
    string_value: ""                  # Acts as a searchable tag
```

---

## Minimal Examples

### Minimal Model (YAML source)

```yaml
source: My Models
models:
  - name: my-org/my-model
    provider: My Organization
    description: A simple text generation model.
    license: apache-2.0
    tasks:
      - text-generation
    artifacts:
      - uri: oci://registry.example.com/my-org/my-model:v1.0
```

### Minimal Model (Hugging Face source)

No data file needed. Configure only the sources file:

```yaml
model_catalogs:
  - name: My HF Models
    id: my_hf_models
    type: hf
    enabled: true
    properties:
      apiKeyEnvVar: "HF_API_KEY"
    includedModels:
      - "meta-llama/Llama-3.2-1B"
      - "microsoft/phi-2"
```

### Minimal MCP Server (Local)

```yaml
source: My Servers
mcp_servers:
  - name: my-mcp-server
    provider: My Organization
    description: A simple MCP server.
    version: "1.0.0"
    license: apache-2.0
    transports:
      - http
    tools:
      - name: hello
        description: Returns a greeting
        accessType: read_only
        parameters: []
    artifacts:
      - uri: oci://registry.example.com/my-org/my-mcp-server:1.0.0
```

### Minimal MCP Server (Remote)

```yaml
source: Remote Servers
mcp_servers:
  - name: remote-api-server
    provider: Cloud Provider
    description: A remotely hosted MCP server.
    deploymentMode: remote
    endpoints:
      http: https://api.example.com/mcp/
    tools:
      - name: query
        description: Execute a query
        accessType: read_only
        parameters:
          - name: input
            type: string
            description: Query input
            required: true
```

---

## Complete Examples

### Full Model with Metrics

```yaml
source: Production Models
models:
  - name: my-org/production-llm-8b
    provider: My Organization
    logo: data:image/svg+xml;base64,<base64-encoded-svg>
    description: |-
      An 8B parameter LLM validated for production deployment.
    readme: |-
      # Production LLM 8B

      Full markdown documentation goes here...
    language: ["en", "es", "fr"]
    license: apache-2.0
    licenseLink: https://www.apache.org/licenses/LICENSE-2.0
    maturity: Production
    libraryName: transformers
    tasks:
      - text-generation
      - question-answering
    validatedTasks:
      - text-generation
      - question-answering
    createTimeSinceEpoch: "1737331200000"
    lastUpdateTimeSinceEpoch: "1737331200000"
    customProperties:
      validated:
        string_value: ""
        metadataType: MetadataStringValue
      model_type:
        string_value: "generative"
        metadataType: MetadataStringValue
    artifacts:
      # Model binary
      - uri: oci://registry.example.com/my-org/llm-8b:v1.0
        createTimeSinceEpoch: "1737331200000"
        lastUpdateTimeSinceEpoch: "1737331200000"
        customProperties:
          architecture:
            metadataType: MetadataStringValue
            string_value: '["amd64"]'

      # Performance benchmark
      - artifactType: metrics-artifact
        metricsType: performance-metrics
        createTimeSinceEpoch: "1736899200000"
        lastUpdateTimeSinceEpoch: "1736899200000"
        customProperties:
          config_id:
            string_value: "llm-8b-h100-2gpu"
            metadataType: MetadataStringValue
          use_case:
            string_value: "chatbot"
            metadataType: MetadataStringValue
          hardware_type:
            string_value: "H100"
            metadataType: MetadataStringValue
          hardware_count:
            int_value: "2"
            metadataType: MetadataIntValue
          ttft_mean:
            double_value: 85.2
            metadataType: MetadataDoubleValue
          tps_mean:
            double_value: 1105.4
            metadataType: MetadataDoubleValue

      # Accuracy benchmark
      - artifactType: metrics-artifact
        metricsType: accuracy-metrics
        createTimeSinceEpoch: "1736899200000"
        lastUpdateTimeSinceEpoch: "1736899200000"
        customProperties:
          benchmark:
            string_value: "mmlu"
            metadataType: MetadataStringValue
          score:
            double_value: 75.8
            metadataType: MetadataDoubleValue
          score_metric:
            string_value: "accuracy_percent"
            metadataType: MetadataStringValue
```

### Full Model with Tool-Calling Configuration

```yaml
source: Tool-Capable Models
models:
  - name: my-org/function-caller-7b
    provider: My Organization
    description: |-
      A 7B parameter model optimized for tool calling and function execution
      with structured JSON output.
    language: ["en"]
    license: mit
    licenseLink: https://opensource.org/licenses/MIT
    libraryName: transformers
    tasks:
      - tool-calling
      - text-generation
    validatedTasks:
      - tool-calling
      - text-generation
    servingConfig:
      toolCalling:
        toolCallParser: granite
        chatTemplate: opt/app-root/template/tool_chat_template_granite.jinja
        enableAutoToolChoice: true
        requiredArgs:
          - "--config_format granite"
    createTimeSinceEpoch: "1738368000000"
    lastUpdateTimeSinceEpoch: "1738368000000"
    customProperties:
      model_type:
        string_value: "agent"
        metadataType: MetadataStringValue
    artifacts:
      - uri: oci://registry.example.com/my-org/function-caller-7b:v1.2
        createTimeSinceEpoch: "1738368000000"
        lastUpdateTimeSinceEpoch: "1738368000000"
```

### Full MCP Server with Runtime Metadata

```yaml
source: Production Servers
mcp_servers:
  - name: my-platform-server
    provider: My Organization
    license: apache-2.0
    license_link: https://www.apache.org/licenses/LICENSE-2.0
    description: >-
      Platform management MCP server with full deployment metadata.
    readme: |-
      # My Platform Server

      Full markdown documentation...
    version: "1.0.0"
    transports:
      - http
    tags:
      - platform
      - management
    logo: data:image/svg+xml;base64,<base64-encoded-svg>
    documentationUrl: https://docs.example.com/mcp
    repositoryUrl: https://github.com/my-org/platform-mcp
    sourceCode: my-org/platform-mcp
    publishedDate: "2025-03-01"
    tools:
      - name: list_resources
        description: List platform resources
        accessType: read_only
        parameters:
          - name: namespace
            type: string
            description: Target namespace
            required: false
      - name: create_resource
        description: Create a new resource
        accessType: read_write
        parameters:
          - name: name
            type: string
            description: Resource name
            required: true
          - name: config
            type: object
            description: Resource configuration
            required: false
    artifacts:
      - uri: oci://registry.example.com/my-org/platform-mcp:1.0.0
        createTimeSinceEpoch: "1740787200000"
        lastUpdateTimeSinceEpoch: "1740787200000"
    runtimeMetadata:
      defaultPort: 8080
      mcpPath: /mcp
      prerequisites:
        serviceAccount:
          required: true
          hint: "Needs 'edit' ClusterRole"
          suggestedName: platform-editor
        secrets:
          - name: platform-credentials
            description: "API credentials for the platform"
            keys:
              - key: api-token
                description: Platform API token
                envVarName: PLATFORM_TOKEN
                required: true
            mountAsFile: false
    securityIndicators:
      verifiedSource: true
      secureEndpoint: true
      sast: true
      readOnlyTools: false
    customProperties:
      platform:
        metadataType: MetadataStringValue
        string_value: ""
      infrastructure:
        metadataType: MetadataStringValue
        string_value: ""
    createTimeSinceEpoch: "1740787200000"
    lastUpdateTimeSinceEpoch: "1740787200000"
```

### Hugging Face Sources with Organization Filtering

```yaml
model_catalogs:
  # All models from a specific organization
  - name: Meta LLaMA Models
    id: hf-meta-llama
    type: hf
    enabled: true
    properties:
      apiKeyEnvVar: "HF_API_KEY"
      allowedOrganization: "meta-llama"
    includedModels:
      - "*"                # All meta-llama models
    excludedModels:
      - "*-GGUF"           # Exclude quantized variants

  # Cherry-pick specific models across organizations
  - name: Curated Models
    id: hf-curated
    type: hf
    enabled: true
    properties:
      apiKeyEnvVar: "HF_API_KEY"
    includedModels:
      - "meta-llama/Llama-3.2-1B"
      - "microsoft/phi-2"
      - "microsoft/phi-3*"       # All phi-3 variants
    excludedModels:
      - "*-base"
      - "*-draft"
```
