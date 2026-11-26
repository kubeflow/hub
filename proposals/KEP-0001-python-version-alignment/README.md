# KEP-0001: Align Python Client Minimum Version with Python Maintenance Status

## Summary

Align the [Model Registry Python client](../../clients/python) minimum supported Python version with the Python Software Foundation's actively maintained versions, in order to ensure timely CVE updates and dependency security patches.

## Motivation

Python versions that have reached End of Life (EOL) no longer receive security updates. Dependencies for EOL Python versions may not receive CVE patches, creating security risks for Model Registry python client users.

### Goals

- Support only actively maintained Python versions (at the time of writing this means Python `3.10+`, as `3.9` reaches EOL October 2025)
- Maintain alignment by updating _minimum_ Python version when versions reach EOL
- Keep dependency security patches current (as defined in [SECURITY.md](../../SECURITY.md))

### Non-Goals

- Changing the MR Python client versioning scheme (at the time of writing `0.X` continues to indicate client API version, not dependency compatibility)
- Supporting Python versions beyond their EOL dates
- Backporting features to older client releases

## Proposal

Update minimum Python version requirement to match the lowest currently maintained Python version per https://devguide.python.org/versions/. When a Python version reaches EOL, update `pyproject.toml` and GitHub Actions workflows accordingly.

The client version (at the time of writing `0.3.X`) remains unchanged, as the `0.X` indicates the client API, not Python dependency constraints.

### Risks and Mitigations

**Risk**: Users on EOL Python versions cannot use new client releases.

**Mitigation**: Previous client versions remain available. Users requiring EOL Python versions can use older client releases, accepting security risks of unmaintained dependencies.

## Design Details

### Test Plan

[x] Existing test suite covers Python version compatibility

#### Unit Tests

Existing unit tests run against all supported Python versions via Nox sessions.

#### E2E tests

Existing E2E tests in [.github/workflows/python-tests.yml](../../.github/workflows/python-tests.yml) validate functionality across Python versions.

### Graduation Criteria

N/A - this is a maintenance policy, not a feature.

## Implementation History

- 2025-12-16: KEP creation and code changes to demonstrate feasibility.

## Drawbacks

Users on EOL Python versions must either upgrade Python or use older client releases.

## Alternatives

1. Continue supporting EOL Python versions indefinitely
   - Not feasible: Creates security debt and maintenance burden for unmaintained dependencies

2. Maintain separate branches for EOL Python versions
   - Not feasible: Excessive maintenance overhead with minimal benefit
