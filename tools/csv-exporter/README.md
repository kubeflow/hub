# Model Catalog CSV Export

A standalone Python CLI script that exports Model Catalog metadata to a CSV file.

## Prerequisites

- Python 3.10 or later
- Network access to the Model Catalog API

No third-party Python packages required — the script uses only the Python standard library.

## Usage

```bash
python model_catalog_export.py --url <catalog-api-url> --output <file.csv>
```

### Arguments

| Argument       | Required | Description                                              |
|----------------|----------|----------------------------------------------------------|
| `--url`          | Yes      | Catalog API base URL (e.g., `http://localhost:8080`)     |
| `--output`       | Yes      | Output CSV file path (not required with `--list-sources`)|
| `--source`       | No       | Filter by source ID (repeatable for multiple sources)    |
| `--limit`        | No       | Maximum number of models to export                       |
| `--page-size`    | No       | Models per API page (default: 100)                       |
| `--header`       | No       | HTTP header as `"Name: Value"` (repeatable)              |
| `--list-sources` | No       | List available sources and exit                          |

### Examples

Export all models:

```bash
python model_catalog_export.py \
  --url http://localhost:8080 \
  --output models.csv
```

List available sources:

```bash
python model_catalog_export.py \
  --url http://localhost:8080 \
  --list-sources
```

```
ID                NAME              STATUS
community         Community         available
validated_models   Validated Models  available
```

Export models from a specific source (use the ID from `--list-sources`):

```bash
python model_catalog_export.py \
  --url http://localhost:8080 \
  --output filtered.csv \
  --source community
```

Export with authentication:

```bash
python model_catalog_export.py \
  --url https://catalog-api.example.com \
  --output models.csv \
  --header "Authorization: Bearer $TOKEN"
```

Limit output to 50 models:

```bash
python model_catalog_export.py \
  --url http://localhost:8080 \
  --output top50.csv \
  --limit 50
```

## Accessing the Model Catalog API

The Model Catalog API is a cluster-internal service. There are several ways to access it
depending on your environment.

### Ingress / Route

If an ingress or route has been created for the catalog service, use the external URL directly:

```bash
python model_catalog_export.py --url "https://catalog.example.com" --output models.csv
```

### Port Forward

Forward the catalog service port to your local machine:

```bash
kubectl port-forward svc/model-catalog -n <namespace> 8080:8080 &
python model_catalog_export.py --url http://localhost:8080 --output models.csv
```

## Authentication

The script does not implement its own authentication flow. Use `--header` to pass
credentials when the API requires authentication:

```bash
# Bearer token
python model_catalog_export.py \
  --url https://catalog-api.example.com \
  --output models.csv \
  --header "Authorization: Bearer $(cat /var/run/secrets/token)"

# Multiple headers
python model_catalog_export.py \
  --url https://catalog-api.example.com \
  --output models.csv \
  --header "Authorization: Bearer $TOKEN" \
  --header "X-Custom-Header: value"
```

When using `kubectl port-forward`, authentication is typically not required since the
connection is already authenticated through your kubeconfig.

## Scheduled Export (Kubernetes CronJob)

The script's CLI interface makes it suitable for use as a Kubernetes CronJob:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: catalog-export
spec:
  schedule: "0 6 * * 1"  # Every Monday at 6am
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: export
            image: python:3.12-slim
            command: ["sh", "-c"]
            args:
            - |
              python /scripts/model_catalog_export.py \
                --url http://model-catalog:8080 \
                --output /exports/models.csv \
                --header "Authorization: Bearer $(cat /var/run/secrets/token)"
            volumeMounts:
            - name: scripts
              mountPath: /scripts
            - name: exports
              mountPath: /exports
            - name: token
              mountPath: /var/run/secrets
          restartPolicy: OnFailure
          volumes:
          - name: scripts
            configMap:
              name: catalog-export-script
          - name: exports
            persistentVolumeClaim:
              claimName: catalog-exports
          - name: token
            secret:
              secretName: catalog-api-token
```

## CSV Format

The output is RFC 4180-compliant CSV with UTF-8 BOM (for Excel compatibility).

### Model Columns

All top-level model fields are included automatically, in a stable default order.
New fields added to the API will appear in the CSV without code changes.

### Custom Property Columns

Each custom property key found across all exported models becomes its own column,
appended after the model columns in alphabetical order. Models that lack a particular
custom property have an empty cell for that column.

### Value Handling

- **Strings, numbers, booleans**: written directly
- **Lists** (language, tasks, validatedTasks): JSON-encoded (e.g., `["en", "es"]`)
- **Custom properties**: scalar values extracted from the metadata wrapper; complex types
  (proto, struct) are JSON-stringified
- **Null or missing fields**: empty cell

### Excluded Fields

The following fields are excluded because they are too large or not useful in
tabular form:

- `readme` — can exceed 32K characters
- `logo` — can contain large data URLs
- `servingConfig` — nested serving configuration object
- `customProperties` — handled separately as dynamic columns (see above)

## Troubleshooting

### Connection refused

```
Error: cannot connect to http://localhost:8080/...: Connection refused
```

The catalog service is not reachable. Check that:
- `kubectl port-forward` is running (if using port-forward)
- The correct namespace and service name are used
- The catalog service pod is running: `kubectl get pods -n <namespace>`

### HTTP 401 Unauthorized

```
Error: HTTP 401 from http://...
```

Authentication is required. Pass a valid bearer token via `--header`:

```bash
--header "Authorization: Bearer $TOKEN"
```

### HTTP 403 Forbidden

The authenticated user does not have permission to access the catalog API.
Check RBAC policies in the target namespace.

### Empty CSV (header only)

The catalog contains no models matching your filters. Verify by checking
the API directly:

```bash
curl -s http://localhost:8080/api/model_catalog/v1alpha1/models | python -m json.tool
```

If using `--source`, confirm the source name is correct.

### Excel shows garbled characters

The CSV includes a UTF-8 BOM for Excel compatibility. If characters still
display incorrectly, open the file via Excel's Data > From Text/CSV import
and select UTF-8 encoding.
