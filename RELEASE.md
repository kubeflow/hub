# Release Process

This document describes the Release process followed by this Kubeflow Hub component project, enacted by its Maintainers.

# Principles

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

# Instructions

These instructions can be followed by the Maintainers with write access on the repository.

Assuming the following remotes are setup locally:

```
origin	git@github.com:<your username>/hub.git (fetch)
origin	git@github.com:<your username>/hub.git (push)
upstream	git@github.com:kubeflow/hub.git (fetch)
upstream	git@github.com:kubeflow/hub.git (push)
```

and for the rest of this instructions, the `<your username>` will be referred as `mr_maintainer`.

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
TDATE=$(date "+%Y%m%d")
```

> [!NOTE]
> We no longer explicits the `-alpha` suffix (see [here](https://github.com/kubeflow/hub/issues/435#issuecomment-2384745910)).

- create the release branch upstream

```
git checkout -b release/$VVERSION
git push upstream release/$VVERSION
```

this creates the release branch upstream.

Create a PR to update what's needed on the release branch, i.e. to update the manifest images.

```
git checkout -b mr_maintainer-$TDATE-upstreamSync
pushd manifests/kustomize/base && kustomize edit set image ghcr.io/kubeflow/hub/server=ghcr.io/kubeflow/hub/server:$VVERSION && popd
pushd manifests/kustomize/options/csi && kustomize edit set image ghcr.io/kubeflow/hub/storage-initializer=ghcr.io/kubeflow/hub/storage-initializer:$VVERSION && popd
pushd manifests/kustomize/options/ui/base && kustomize edit set image model-registry-ui=ghcr.io/kubeflow/hub/ui:$VVERSION && popd
pushd manifests/kustomize/options/catalog/base && kustomize edit set image ghcr.io/kubeflow/hub/server=ghcr.io/kubeflow/hub/server:$VVERSION && popd
git add .
git commit -s

# suggested commit msg: "chore: align manifest for 0.2.10"

# using `git push origin`
# will give back convenient command on the screen for copy/paste:
# eg: git push --set-upstream origin mr_maintainer-20241108-upstreamSync
git push --set-upstream origin mr_maintainer-$TDATE-upstreamSync
```

- create PR ⚠️ targeting the _release branch_ ⚠️ specifically (title ~like: `chore: align manifest for 0.2.10`)
- merge the PR (you can manually add the approved, lgtm labels)

- optional. if you create the tag from local git (see point below); await GHA complete that push Container images to docker.io or any other KF registry: https://github.com/kubeflow/hub/actions
- create [the Release from GitHub](https://github.com/kubeflow/hub/releases/new), ⚠️ select the _release branch_ ⚠️ , input the _new tag_<br/>(in this example the tag is created from GitHub; alternatively, you could just do the tag manually by checking out the release branch locally--remember to pull!!--and issuing the tag from local machine).
- encouraging to use the "alpha" version policy of KF in the beginning of the release markdown (see previous releases).

It is helpful to prefix this in the release notes:

```md
> **Alpha**
> This Kubeflow component has alpha status with limited support. See the [Kubeflow versioning policies](https://www.kubeflow.org/docs/started/support/#application-status). The Kubeflow team is interested in your [feedback](https://github.com/kubeflow/hub/issues/new/choose) about the usability of the feature.
```

- release the MR Python client

```
git checkout release/$VVERSION
git pull upstream release/$VVERSION
git tag py-$VVERSION
git push upstream py-$VVERSION
```

- add a Tag for the pkg/openapi

```
git checkout release/$VVERSION
git pull upstream release/$VVERSION
git tag pkg/openapi/$VVERSION
git push upstream pkg/openapi/$VVERSION
```

- add a Tag for the pkg/inferenceservice-controller

```
git checkout release/$VVERSION
git pull upstream release/$VVERSION
git tag pkg/inferenceservice-controller/$VVERSION
git push upstream pkg/inferenceservice-controller/$VVERSION
```

At this point, a release as been created, both the container images and the Python client on pypi.

## KF/manifests

The KF/hub manifests need to be sync'd to KF/manifests repository using the Manifest/Platform WG provided script (in the KF/manifests repo).

Example PR:
- https://github.com/kubeflow/manifests/pull/3053

It is supposed to work by leveraging sync script in KF/manifests repo:
- https://github.com/pboyd/kubeflow-manifests/blob/163b45fcb55bdcbafa33a843a856a6bd96c5cecd/scripts/synchronize-hub-manifests.sh

## KF/website

Update latest MR release version number in the KF/website repo.

Example PR:
- https://github.com/kubeflow/website/pull/4046

Please notice the OpenAPI spec in the Reference section is automatically updated, since it is sourced from the repo: https://github.com/kubeflow/website/blob/b97081e8e19a06430268e1fa9a38808f2a04cf69/content/en/docs/components/hub/reference/rest-api.md?plain=1#L44-L46

## Anticipate prerequisites

See at the beginning "Prerequisites", to facilitate the next round, now it's the best time:
- bump already MR py client to the next version, example PR

https://github.com/kubeflow/hub/pull/871
