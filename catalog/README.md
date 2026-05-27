# Model Catalog Service

The Model Catalog Service provides a **read-only discovery service** for ML models across multiple catalog sources. It acts as a federated metadata aggregation layer, allowing users to search and discover models from various external catalogs through a unified REST API.

## Architecture Overview

The catalog service operates as a **metadata aggregation layer** that:
- Federates model discovery across different external catalogs
- Provides a unified REST API for model search and discovery
- Uses pluggable source providers for extensibility
- Operates without traditional database storage (file-based configuration)

### Supported Catalog Sources

- **YAML Catalog** - Static YAML files containing model metadata
- **Hugging Face Hub** - Discover models from Hugging Face's model repository

## REST API

### Base URL
`/api/model_catalog/v1alpha1`

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/sources` | List all catalog sources with pagination |
| `GET` | `/models` | Search models across sources (requires `source` parameter) |
| `GET` | `/sources/{source_id}/models/{model_name+}` | Get specific model details |
| `GET` | `/sources/{source_id}/models/{model_name}/artifacts` | List model artifacts |

### OpenAPI Specification

View the complete API specification:
- [Swagger UI](https://www.kubeflow.org/docs/components/model-registry/reference/model-catalog-rest-api/#swagger-ui)
- [Swagger Playground](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/kubeflow/hub/main/api/openapi/catalog.yaml)

## Data Models

### CatalogSource
Simple source metadata:
```json
{
  "id": "string",
  "name": "string"
}
```

### CatalogModel
Rich model metadata including:
- Basic info: `name`, `description`, `readme`, `maturity`
- Technical: `language[]`, `tasks[]`, `libraryName`
- Legal: `license`, `licenseLink`, `provider`
- Extensible: `customProperties` (key-value metadata)

### CatalogModelArtifact
Artifact references:
```json
{
  "uri": "string",
  "customProperties": {}
}
```

## Custom Properties

Custom properties provide extensible metadata for models and artifacts beyond the predefined schema fields. They enable storing domain-specific metadata, classification tags, and arbitrary key-value data.

### Overview

Custom properties can be attached to:
- **CatalogModel**: Model-level metadata (e.g., model type, validation status)
- **CatalogModelArtifact**: Artifact-level metadata (e.g., validation date, deployment targets)
- **CatalogMetricsArtifact**: Metrics metadata (e.g., benchmark names, hardware configurations)

Each custom property consists of:
- **Key**: Property name (string)
- **Value**: Typed metadata value with one of the following types:
  - `MetadataStringValue`: String values
  - `MetadataIntValue`: Integer values
  - `MetadataDoubleValue`: Floating-point values
  - `MetadataBoolValue`: Boolean values

### Model Type Property

The `model_type` custom property is a standardized property for categorizing models by their AI/ML paradigm. It enables filtering and governance based on model characteristics.

#### Specification

**Property Name**: `model_type`

**Metadata Type**: `MetadataStringValue`

**Allowed Values**:
- `predictive` - Traditional ML models (regression, classification, forecasting, clustering, etc.)
- `generative` - Generative AI models (LLMs, diffusion models, GANs, VAEs, etc.)
- `unknown` - Model type not yet determined or not applicable

#### Usage

The `model_type` property should be set as a custom property on model artifacts to indicate the model's category:

**YAML Format** (for YAML catalog sources):
```yaml
models:
  - name: my-regression-model
    description: Sales forecasting model
    customProperties:
      model_type:
        metadataType: MetadataStringValue
        string_value: "predictive"
    artifacts:
      - uri: oci://registry.example.com/models/sales-forecast:v1.0

  - name: my-llm-model
    description: Large language model for text generation
    customProperties:
      model_type:
        metadataType: MetadataStringValue
        string_value: "generative"
    artifacts:
      - uri: oci://registry.example.com/models/text-generator:v2.0
```

**REST API Response** (JSON):
```json
{
  "name": "my-regression-model",
  "description": "Sales forecasting model",
  "customProperties": {
    "model_type": {
      "metadataType": "MetadataStringValue",
      "string_value": "predictive"
    }
  }
}
```

#### Model Type Classification Guide

**Predictive Models** (`predictive`):
- Regression models (linear, polynomial, etc.)
- Classification models (logistic regression, SVM, random forest, etc.)
- Time-series forecasting
- Clustering algorithms
- Anomaly detection
- Traditional neural networks (CNNs for classification, RNNs for prediction)
- Gradient boosting models (XGBoost, LightGBM, CatBoost)
- Recommendation systems (collaborative filtering)

**Generative Models** (`generative`):
- Large Language Models (LLMs) - GPT, BERT, Llama, etc.
- Text-to-image models - Stable Diffusion, DALL-E, etc.
- Generative Adversarial Networks (GANs)
- Variational Autoencoders (VAEs)
- Diffusion models
- Text-to-speech and speech-to-text models
- Code generation models
- Transformer-based generation models

**Unknown** (`unknown`):
- Hybrid models that combine both paradigms
- Experimental models under development
- Models where classification is not yet determined

#### Automatic Classification for Hugging Face Models

The Hugging Face catalog provider automatically classifies models based on their task types. When a model is imported from Hugging Face, the `model_type` custom property is set based on the model's declared tasks using a heuristic classification system.

**Classification Heuristic:**

The classification logic examines the model's task list and:
1. Checks if any task matches the **generative tasks** set → classifies as `"generative"`
2. Checks if any task matches the **predictive tasks** set → classifies as `"predictive"`
3. If both generative and predictive tasks are present, **generative takes priority**
4. If no matching tasks are found → classifies as `"unknown"`

**Task Mappings:**

The following Hugging Face task types are mapped to model types:

**Generative Tasks** (maps to `"generative"`):
- `text-generation` - Text generation models
- `summarization` - Text summarization models
- `translation` - Translation models
- `text-to-image` - Text-to-image generation models
- `unconditional-image-generation` - Image generation models
- `image-to-image` - Image-to-image transformation models
- `text-to-speech` - Text-to-speech synthesis models
- `audio-to-audio` - Audio transformation models

**Predictive Tasks** (maps to `"predictive"`):
- `text-classification` - Text classification models
- `image-classification` - Image classification models
- `zero-shot-classification` - Zero-shot classification models
- `audio-classification` - Audio classification models
- `question-answering` - Question answering models
- `document-question-answering` - Document QA models
- `object-detection` - Object detection models
- `image-segmentation` - Image segmentation models
- `keypoint-detection` - Keypoint detection models
- `feature-extraction` - Feature extraction models
- `image-feature-extraction` - Image feature extraction models
- `fill-mask` - Masked language models

**Note:** The classification is performed automatically when models are fetched from Hugging Face. Models that don't match any known tasks will be classified as `"unknown"` and can be manually updated via the YAML catalog or API if needed.

### Querying and Filtering by Custom Properties

#### Filter by Model Type

Search for all generative AI models:
```bash
GET /api/model_catalog/v1alpha1/models?source=my-catalog&filterQuery=model_type.string_value='generative'
```

Search for predictive models:
```bash
GET /api/model_catalog/v1alpha1/models?source=my-catalog&filterQuery=model_type.string_value='predictive'
```

#### Combining Filters

Filter by model type and other criteria:
```bash
# Generative models with production maturity
GET /api/model_catalog/v1alpha1/models?source=my-catalog&filterQuery=model_type.string_value='generative' AND maturity='Production'

# Predictive models for specific tasks
GET /api/model_catalog/v1alpha1/models?source=my-catalog&filterQuery=model_type.string_value='predictive' AND tasks CONTAINS 'regression'
```

### Additional Custom Properties Examples

#### Validation and Certification

```yaml
customProperties:
  validated:
    metadataType: MetadataStringValue
    string_value: ""
  validation_status:
    metadataType: MetadataStringValue
    string_value: "certified"
  validation_date:
    metadataType: MetadataStringValue
    string_value: "2025-01-20"
  compliance:
    metadataType: MetadataStringValue
    string_value: "GDPR,CCPA,SOC2"
```

#### Performance and Hardware

```yaml
customProperties:
  hardware_type:
    metadataType: MetadataStringValue
    string_value: "H100"
  hardware_count:
    metadataType: MetadataIntValue
    int_value: "2"
  throughput_tps:
    metadataType: MetadataDoubleValue
    double_value: 1105.4
  latency_p95_ms:
    metadataType: MetadataDoubleValue
    double_value: 108.3
```

#### Deployment Metadata

```yaml
customProperties:
  deployment_type:
    metadataType: MetadataStringValue
    string_value: "production"
  framework_type:
    metadataType: MetadataStringValue
    string_value: "vllm"
  framework_version:
    metadataType: MetadataStringValue
    string_value: "v0.8.4"
  use_case:
    metadataType: MetadataStringValue
    string_value: "chatbot"
```

### Best Practices

1. **Use Standardized Properties**: For common use cases like `model_type`, use the documented property names and values to ensure consistency across catalogs.

2. **Choose Appropriate Types**: Select the correct metadata type for your values:
   - Use `MetadataStringValue` for text, enums, and identifiers
   - Use `MetadataIntValue` for counts and whole numbers
   - Use `MetadataDoubleValue` for measurements and metrics
   - Use `MetadataBoolValue` for flags

3. **Document Custom Properties**: Maintain documentation for any custom properties specific to your organization or use case.

4. **Validate Values**: When using enum-like properties (like `model_type`), validate values against the allowed set to prevent inconsistencies.

5. **Use Hierarchical Keys**: For complex metadata, consider using dot-notation or underscores to create logical groupings (e.g., `validation_status`, `hardware_type`).

## Configuration

The catalog service uses **file-based configuration** instead of traditional databases:

```yaml
# catalog-sources.yaml
catalogs:
  - id: "yaml-catalog"
    name: "Local YAML Catalog"
    type: "yaml"
    properties:
      path: "./models"
```

### Hugging Face Source Configuration

The Hugging Face catalog source allows you to discover and import models from the Hugging Face Hub. To configure a Hugging Face source:

#### 1. Set Your API Key

Setting a Hugging Face API key is optional. Hugging Face  requires an API key for authentication for full access to data of models that are private and/or gated. If an API key is NOT set, private models will be entirely unavailable and gated models will have limited metadata. By default, the service reads the API key from the `HF_API_KEY` environment variable:

**Getting a Hugging Face API Key:**
1. Sign up or log in to [Hugging Face](https://huggingface.co)
2. Go to your [Settings > Access Tokens](https://huggingface.co/settings/tokens)
3. Create a new token with "Read" permissions
4. Copy the token and set it as an environment variable

**For Kubernetes deployments:**
- Store the API key in a Kubernetes Secret
- Reference it in your deployment configuration
- The catalog service will read it from the configured environment variable (defaults to `HF_API_KEY`)

```bash
kubectl create secret generic model-catalog-hf-api-key \
  --from-literal=HF_API_KEY="your-api-key-here" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl rollout restart deployment model-catalog-server -n kubeflow
```

**Custom Environment Variable Name:**
You can configure a custom environment variable name per source by setting the `apiKeyEnvVar` property in your source configuration (see below). This is useful when you need different API keys for different sources.

**Important Notes:**
- **Private Models**: For private models, the API key must belong to an account that has been granted access to the model. Without proper access, the catalog service will not be able to retrieve model information.
- **Gated Models**: For gated models (models with usage restrictions), you must accept the model's terms of service on Hugging Face before the catalog service can access all available model information. Visit the model's page on Hugging Face and accept the terms to ensure full metadata is available.

#### 2. Configure the Source

Add a Hugging Face source to your `catalog-sources.yaml`:

```yaml
catalogs:
  - name: "Hugging Face Hub"
    id: "huggingface"
    type: "hf"
    enabled: true
    # Required: List of model identifiers to include
    # Format: "organization/model-name" or "username/model-name"
    # Supports wildcard patterns: "organization/*" or "organization/prefix*"
    includedModels:
      - "meta-llama/Llama-3.1-8B-Instruct"
      - "microsoft/phi-2"
      - "microsoft/phi-3*"  # All models starting with "phi-3"

    # Optional: Exclude specific models or patterns
    # Supports exact matches or patterns ending with "*"
    excludedModels:
      - "some-org/unwanted-model"
      - "another-org/test-*"  # Excludes all models starting with "test-"

    # Optional: Configure a custom environment variable name for the API key
    # Defaults to "HF_API_KEY" if not specified
    properties:
      apiKeyEnvVar: "MY_CUSTOM_API_KEY_VAR"
```

#### Organization-Restricted Sources

You can restrict a source to only fetch models from a specific organization using the `allowedOrganization` property. This automatically prefixes all model patterns with the organization name:

```yaml
catalogs:
  - name: "Meta LLaMA Models"
    id: "meta-llama-models"
    type: "hf"
    enabled: true
    properties:
      allowedOrganization: "meta-llama"
      apiKeyEnvVar: "HF_API_KEY"
    includedModels:
      # These patterns are automatically prefixed with "meta-llama/"
      - "*"           # Expands to: meta-llama/*
      - "Llama-3*"    # Expands to: meta-llama/Llama-3*
      - "CodeLlama-*" # Expands to: meta-llama/CodeLlama-*
    excludedModels:
      - "*-4bit"      # Excludes: meta-llama/*-4bit
      - "*-GGUF"      # Excludes: meta-llama/*-GGUF
```

**Benefits of organization-restricted sources:**
- **Simplified configuration**: No need to repeat organization name in every pattern
- **Security**: Prevents accidental inclusion of models from other organizations
- **Convenience**: Use `"*"` to get all models from an organization
- **Performance**: Optimized API calls when fetching from a single organization

#### Model Filtering

Both `includedModels` and `excludedModels` are top-level properties (not nested under `properties`):

- **`includedModels`** (required): List of model identifiers to fetch from Hugging Face
- **`excludedModels`** (optional): List of models or patterns to exclude from the results

#### Supported Pattern Types

**Exact Model Names:**
```yaml
includedModels:
  - "meta-llama/Llama-3.1-8B-Instruct"  # Specific model
  - "microsoft/phi-2"                    # Specific model
```

**Wildcard Patterns:**

In `includedModels`, wildcards can match model names by a prefix.

```yaml
includedModels:
  - "microsoft/phi-*"      # All models starting with "phi-"
  - "meta-llama/Llama-3*"  # All models starting with "Llama-3"
  - "huggingface/*"        # All models from huggingface organization
```

**Organization-Only Patterns (with `allowedOrganization`):**
```yaml
properties:
  allowedOrganization: "meta-llama"
includedModels:
  - "*"          # All models from meta-llama organization
  - "Llama-3*"   # All meta-llama models starting with "Llama-3"
  - "CodeLlama-*" # All meta-llama models starting with "CodeLlama-"
```

#### Pattern Validation

**Valid patterns:**
- `"org/model"` - Exact model name
- `"org/prefix*"` - Models starting with prefix
- `"org/*"` - All models from organization
- `"*"` - All models (only when using `allowedOrganization`)

**Invalid patterns (will be rejected):**
- `"*"` - Global wildcard (without `allowedOrganization`)
- `"*/*"` - Global organization wildcard
- `"org*"` - Wildcard in organization name
- `"org/"` - Empty model name
- `"*prefix*"` - Multiple wildcards

#### Exclusion Patterns

The `excludedModels` property supports prefixes like `includedModels` and also suffixes and mid-name wildcards:
- **Exact matches**: `"meta-llama/Llama-3.1-8B-Instruct"` - excludes this specific model
- **Pattern matching**:
    - `"*-draft"` - excludes all models ending with "-draft"
    - `"Llama-3.*-Instruct"` - excludes all Llama 3.x models ending with "-Instruct"
- **Organization patterns**: `"test-org/*"` - excludes all models from test-org

## Development

### Prerequisites
- Go >= 1.26
- Java >= 11.0 (for OpenAPI generation)
- Node.js >= 20.0.0 (for GraphQL schema downloads)

### Building

Generate OpenAPI server code:
```bash
make gen/openapi-server
```

Generate OpenAPI client code:
```bash
make gen/openapi
```

### Project Structure

```
catalog/
├── cmd/                    # Main application entry point
├── internal/
│   ├── catalog/           # Core catalog logic and providers
│   │   ├── genqlient/     # GraphQL client generation
│   │   └── testdata/      # Test fixtures
│   └── server/openapi/    # REST API implementation
├── pkg/openapi/           # Generated OpenAPI client
├── scripts/               # Build and generation scripts
└── Makefile              # Build targets
```

### Adding New Catalog Providers

1. Implement the `CatalogSourceProvider` interface:
```go
type CatalogSourceProvider interface {
    GetModel(ctx context.Context, name string) (*model.CatalogModel, error)
    ListModels(ctx context.Context, params ListModelsParams) (model.CatalogModelList, error)
    GetArtifacts(ctx context.Context, name string) (*model.CatalogModelArtifactList, error)
}
```

2. Register your provider:
```go
catalog.RegisterCatalogType("my-catalog", func(source *Source) (CatalogSourceProvider, error) {
    return NewMyCatalogProvider(source)
})
```

### Testing

The catalog service includes comprehensive testing:
- Unit tests for core catalog logic
- Integration tests for provider implementations
- OpenAPI contract validation

### Configuration Hot Reloading

The service automatically reloads configuration when the catalog sources file changes, enabling dynamic catalog updates without service restarts.

## Integration

The catalog service is designed to complement the main Model Registry service by providing:
- External model discovery capabilities
- Unified metadata aggregation
- Read-only access to distributed model catalogs

For complete Kubeflow Hub documentation, see the [main README](../README.md).
