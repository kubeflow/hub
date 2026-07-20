# Release Process

This document describes the Release process followed by this Kubeflow Hub component project, enacted by its Maintainers.

## Principles

The Kubeflow Hub follows the [Github Release Workflow](https://github.com/kubeflow/hub/releases), and performs periodic releases in accordance with the Kubeflow Platform WG recommendations.

The Kubeflow Hub follows [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`.

The Kubeflow Hub per governance of the Kubeflow Community, Kubeflow Platform WG, and KSC, releases as Alpha version, including the following statement:

```md
> **Alpha**
> This Kubeflow component has alpha status with limited support. See the [Kubeflow versioning policies](https://www.kubeflow.org/docs/started/support/#application-status). The Kubeflow team is interested in your [feedback](https://github.com/kubeflow/hub/issues/new/choose) about the usability of the feature.
```

The Release of the Kubeflow Hub provides:
- a container image for the Backend; known as the "KF MR Go REST server"
- a Python client to be used in a Jupyter notebook, programmatically, or that can be integrated in the Kubeflow SDK; known as the "MR py client"
- an optional Model Registry Custom Storage Initializer container image for KServe; the "Model Registry CSI"
- a collection of Kubernetes Manifests, which get synchronized to the `kubeflow/manifests` repository
- an update to the Kubeflow website

## Instructions

These instructions can be followed by the Maintainers with write access on the repository.

Assuming the following remotes are setup locally:

```
origin	git@github.com:<your username>/hub.git (fetch)
origin	git@github.com:<your username>/hub.git (push)
upstream	git@github.com:kubeflow/hub.git (fetch)
upstream	git@github.com:kubeflow/hub.git (push)
```

Prerequisites:
- on main branch, the version indicated by the [pyproject.toml](https://github.com/kubeflow/hub/blob/d2312907025adbe83d3faafbecf1474824d055ed/clients/python/pyproject.toml#L3) and [metadadata](https://github.com/kubeflow/hub/blob/d2312907025adbe83d3faafbecf1474824d055ed/clients/python/src/model_registry/__init__.py#L3) of the Model Registry Python client is current (that is, is already valorized to the _target_ release number).
- the main branch is up-to-date, all the required work has been already merged.

```
git checkout main
git pull upstream main
```

Example for `0.2.10` release:

```sh
VVERSION=v0.2.10
```

## Prepare the release branch

```
git switch main
git switch -c release/$VVERSION
git push upstream release/$VVERSION
```

This creates the release branch upstream.

Create a PR to update what's needed on the release branch, i.e. to update the manifest images.

```
git switch -c $VVERSION-manifests
pushd manifests/kustomize/base && kustomize edit set image ghcr.io/kubeflow/hub/server=ghcr.io/kubeflow/hub/server:$VVERSION && popd
pushd manifests/kustomize/options/csi && kustomize edit set image ghcr.io/kubeflow/hub/storage-initializer=ghcr.io/kubeflow/hub/storage-initializer:$VVERSION && popd
pushd manifests/kustomize/options/ui/base && kustomize edit set image model-registry-ui=ghcr.io/kubeflow/hub/ui:$VVERSION && popd
pushd manifests/kustomize/options/catalog/base && kustomize edit set image ghcr.io/kubeflow/hub/server=ghcr.io/kubeflow/hub/server:$VVERSION && popd
VERSION=$(echo $VVERSION | cut -b 2-)
sed -i "s/^\(version = \"\)[^\"]*\"/\1$VERSION\"/" clients/python/pyproject.toml
sed -i "s/^\(__version__ = \"\)[^\"]*\"/\1$VERSION\"/" clients/python/src/model_registry/__init__.py
git commit -asm "chore: update manifests for release $VVERSION"
```

**Note:** On mac, replace `sed -i` with `sed -i ''`.

Look at the changes in the last commit (`git show`) and verify that they are correct.

Create a PR targeting the release branch. If the GitHub CLI is configured: `gh pr create --base release/$VVERSION` Otherwise push the branch and create the PR in the usual way.

Wait for the build to pass, then merge the PR (you can manually add the approved, lgtm labels).

**IMPORTANT:** After the merge, update your local branch:

```
git switch release/$VVERSION
git pull
```

## Create the release

Create [the Release from GitHub](https://github.com/kubeflow/hub/releases/new):
- **Tag:** _Create new tag_, then enter the new version (with the `v`: `v0.3.14`)
- **Target:** ⚠️ select the _release branch_ ⚠️
- **Release title:** enter the version number (again, with the `v`)
- **Release notes:** select the previous release, then "Generate Release Notes" (don't trust *Auto*!)

Paste the following at the beginning of the release notes:

```md
> **Alpha**
> This Kubeflow component has alpha status with limited support. See the [Kubeflow versioning policies](https://www.kubeflow.org/docs/started/support/#application-status). The Kubeflow team is interested in your [feedback](https://github.com/kubeflow/hub/issues/new/choose) about the usability of the feature.
```

Push **Save Draft** (releases are immutable, we're not ready to publish yet).

## Tag the release

Create the tag from the release branch:

```
git switch release/$VVERSION
git pull
git tag $VVERSION -m $VVERSION
git push origin $VVERSION
```

Watch the build progress from Git Hub: _https://github.com/kubeflow/hub/tree/vX.Y.Z_ (`xdg-open https://github.com/kubeflow/hub/tree/$VVERSION`).

The GHA will attach SBOM artifacts to the draft release.

**Wait for the build to finish before continuing.**

## Publish the release

Return to the draft release from the [releases](https://github.com/kubeflow/hub/releases) page, and publish the release.

## Create the other tags

Create tags for the Python client, `pkg/openapi` and the Inference Controller:

```
git switch release/$VVERSION
git pull upstream release/$VVERSION
git tag py-$VVERSION -m py-$VVERSION
git tag pkg/openapi/$VVERSION -m pkg/openapi/$VVERSION
git tag pkg/inferenceservice-controller/$VVERSION -m pkg/inferenceservice-controller/$VVERSION
git push upstream py-$VVERSION pkg/openapi/$VVERSION pkg/inferenceservice-controller/$VVERSION
```

At this point, a release as been created, both the container images and the Python client on pypi.

## `kubeflow/community-distribution`

The `kubeflow/hub` manifests need to be sync'd to `kubeflow/community-distribution` repository.

From a local clone of `community-distribution`:

```
git switch master
git pull
sed -i "s/^\(COMMIT=\"\)[^\"]*\"/\1$VVERSION\"/" scripts/synchronize-hub-manifests.sh
./scripts/synchronize-hub-manifests.sh
```

**Note:** As above, on Mac, use `sed -i ''`

This will create a new branch and make a commit with the updated manifests. Verify the changes with `git show`, then create the PR: `gh pr create` (or push the branch and make a PR using your usual method).

## `kubeflow/website`

From a local close of `kubeflow/website`:

```
git switch master
git pull
echo $VVERSION >layouts/shortcodes/hub/latest-version.html
git commit -asm "hub: bump hub version to $VVERSION"
```

Then create a PR: `gh pr create` (or however you normally make a PR).

Please notice the OpenAPI spec in the Reference section is automatically updated, since it is sourced from the repo: https://github.com/kubeflow/website/blob/b97081e8e19a06430268e1fa9a38808f2a04cf69/content/en/docs/components/hub/reference/rest-api.md?plain=1#L44-L46

## Anticipate prerequisites

See at the beginning "Prerequisites", to facilitate the next round, now it's the best time:
- bump already MR py client to the next version, example PR

https://github.com/kubeflow/hub/pull/871
