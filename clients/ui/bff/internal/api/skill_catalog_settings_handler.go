package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SkillCatalogSettingsSourceConfigEnvelope Envelope[*models.SkillCatalogSourceConfig, None]
type SkillCatalogSettingsSourceConfigListEnvelope Envelope[*models.SkillCatalogSourceConfigList, None]
type SkillCatalogSourcePayloadEnvelope Envelope[*models.SkillCatalogSourceConfigPayload, None]

// skillCatalogNamespace returns the namespace to use for skill catalog ConfigMap reads/writes.
// In dev mode with a catalog namespace configured, that namespace is used so the catalog service
// (which may be in a different namespace than the user) can see the changes.
// In production, user namespace equals catalog namespace, so the user namespace is always correct.
func (app *App) skillCatalogNamespace(userNamespace string) string {
	if app.config.DevMode && app.config.DevModeCatalogNamespace != "" {
		return app.config.DevModeCatalogNamespace
	}
	return userNamespace
}

// pendingSkillCredential is a git token that belongs in the shared git-credentials
// Secret once the source it belongs to has actually been persisted.
type pendingSkillCredential struct {
	key   string
	token string
}

// prepareSkillRepoCredentials moves any admin-supplied git token out of the payload
// and points the repository at where that token will live, via credentialRef. It does
// not write the Secret: the token is returned for commitSkillRepoCredential to store
// once the source has been persisted.
//
// The catalog service has no Kubernetes API access: it resolves credentialRef as a
// filename inside the Secret mounted at /etc/skill-catalog/git-credentials (see
// catalog/internal/catalog/skillcatalog/credentials.go), reading it through
// GIT_ASKPASS for every clone. So the value written to the ConfigMap must be the
// *key* within that one Secret, not a Secret name, and credentialRef has to be set
// before the ConfigMap is written. kubelet refreshes the mount in place, so a newly
// added key becomes readable without restarting the catalog pod.
//
// The token itself is cleared from the payload here so it can never reach the
// ConfigMap or a response body.
func (app *App) prepareSkillRepoCredentials(
	sourceID string,
	payload *models.SkillCatalogSourceConfigPayload,
) (*pendingSkillCredential, error) {
	// A source carries a single repository (validateSkillRepositories enforces it), so
	// there is at most one token. This runs before that validation, though — the token
	// must leave the payload before anything can persist it — so the extra repositories
	// are still rejected here rather than assuming index 0.
	tokenIdx := -1
	for i := range payload.Repositories {
		tok := payload.Repositories[i].AuthToken
		if tok != nil && *tok != "" {
			if tokenIdx >= 0 {
				return nil, fmt.Errorf("%w: a source accepts a single repository, so only one auth token",
					repositories.ErrSkillCatalogValidationFailed)
			}
			tokenIdx = i
		}
	}

	if tokenIdx < 0 {
		// No new token: clear the write-only field and keep any existing
		// credentialRef the client round-tripped from a previous read.
		for i := range payload.Repositories {
			payload.Repositories[i].AuthToken = nil
		}
		return nil, nil
	}

	token := *payload.Repositories[tokenIdx].AuthToken
	for i := range payload.Repositories {
		payload.Repositories[i].AuthToken = nil
	}
	ref := sourceID
	payload.Repositories[tokenIdx].CredentialRef = &ref

	return &pendingSkillCredential{key: sourceID, token: token}, nil
}

// commitSkillRepoCredential stores a prepared token in the shared git-credentials
// Secret. It runs only after the source has been persisted, so a request that is
// rejected — a duplicate id, a failed validation, an edit to a read-only default —
// leaves the Secret untouched. Writing before that check would let a rejected create
// overwrite the token of the existing source whose id it collided with, breaking a
// working source through a request the admin saw fail.
func (app *App) commitSkillRepoCredential(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	pending *pendingSkillCredential,
) error {
	if pending == nil {
		return nil
	}

	secretName := app.config.SkillCatalogGitCredentialsSecret
	if err := client.PatchSecret(ctx, namespace, secretName, map[string]string{pending.key: pending.token}); err != nil {
		// Only a missing Secret is recoverable by creating it. Falling back on any error
		// turns an RBAC or conflict failure into a confusing "already exists" from the
		// create below, hiding the real cause from the operator.
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to store git token in secret %q: %w", secretName, err)
		}
		// The shared Secret may not exist yet on a fresh install; create it with this key.
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			StringData: map[string]string{pending.key: pending.token},
		}
		if _, createErr := client.CreateSecret(ctx, namespace, secret); createErr != nil {
			return fmt.Errorf("failed to store git token in secret %q: %w", secretName, createErr)
		}
	}

	return nil
}

func (app *App) GetAllSkillCatalogSourceConfigsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	configs, err := app.repositories.SkillCatalogSettingsRepository.GetAllSkillCatalogSourceConfigs(ctx, client, app.skillCatalogNamespace(namespace))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err = app.WriteJSON(w, http.StatusOK, SkillCatalogSettingsSourceConfigListEnvelope{Data: configs}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetSkillCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	sourceID := ps.ByName(CatalogSourceId)

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	config, err := app.repositories.SkillCatalogSettingsRepository.GetSkillCatalogSourceConfig(ctx, client, app.skillCatalogNamespace(namespace), sourceID)
	if err != nil {
		if errors.Is(err, repositories.ErrSkillCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err = app.WriteJSON(w, http.StatusOK, SkillCatalogSettingsSourceConfigEnvelope{Data: config}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) CreateSkillCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	var envelope SkillCatalogSourcePayloadEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %w", err))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing required field: data"))
		return
	}

	payload := *envelope.Data
	catalogNS := app.skillCatalogNamespace(namespace)
	pendingCredential, err := app.prepareSkillRepoCredentials(payload.Id, &payload)
	if err != nil {
		if errors.Is(err, repositories.ErrSkillCatalogValidationFailed) {
			app.badRequestResponse(w, r, err)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	created, err := app.repositories.SkillCatalogSettingsRepository.CreateSkillCatalogSourceConfig(ctx, client, catalogNS, payload)
	if err != nil {
		if errors.Is(err, repositories.ErrSkillCatalogSourceAlreadyExist) ||
			errors.Is(err, repositories.ErrSkillCatalogSourceIdRequired) ||
			errors.Is(err, repositories.ErrSkillCatalogValidationFailed) {
			app.badRequestResponse(w, r, err)
		} else if errors.Is(err, repositories.ErrSkillCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Only now that the source exists is its token safe to store: a rejected create
	// above must not touch the Secret, or a duplicate id would overwrite the token of
	// the source it collided with.
	if err := app.commitSkillRepoCredential(ctx, client, catalogNS, pendingCredential); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf(
			"source %q was created, but storing its git token failed; re-save the source to retry: %w",
			payload.Id, err))
		return
	}

	result := SkillCatalogSettingsSourceConfigEnvelope{Data: created}
	w.Header().Set("Location", r.URL.JoinPath(result.Data.Id).String())
	if err = app.WriteJSON(w, http.StatusCreated, result, nil); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON"))
	}
}

func (app *App) UpdateSkillCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	var envelope SkillCatalogSourcePayloadEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %w", err))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing required field: data"))
		return
	}

	sourceID := ps.ByName(CatalogSourceId)
	if sourceID == "" {
		sourceID = envelope.Data.Id
	}

	updatePayload := *envelope.Data
	catalogNS := app.skillCatalogNamespace(namespace)
	pendingCredential, err := app.prepareSkillRepoCredentials(sourceID, &updatePayload)
	if err != nil {
		if errors.Is(err, repositories.ErrSkillCatalogValidationFailed) {
			app.badRequestResponse(w, r, err)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	updated, err := app.repositories.SkillCatalogSettingsRepository.UpdateSkillCatalogSourceConfig(ctx, client, catalogNS, sourceID, updatePayload)
	if err != nil {
		if errors.Is(err, repositories.ErrSkillCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else if errors.Is(err, repositories.ErrSkillCatalogValidationFailed) {
			app.badRequestResponse(w, r, err)
		} else if errors.Is(err, repositories.ErrSkillCatalogCannotChangeDefault) ||
			errors.Is(err, repositories.ErrSkillCatalogCannotChangeType) {
			app.forbiddenResponse(w, r, err.Error())
		} else if errors.Is(err, repositories.ErrSkillCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// As on create: the token is stored only once the update has been accepted, so a
	// 403 on a read-only default or a 404 on a missing source leaves the Secret alone.
	if err := app.commitSkillRepoCredential(ctx, client, catalogNS, pendingCredential); err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf(
			"source %q was updated, but storing its git token failed; re-save the source to retry: %w",
			sourceID, err))
		return
	}

	if err = app.WriteJSON(w, http.StatusOK, SkillCatalogSettingsSourceConfigEnvelope{Data: updated}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) DeleteSkillCatalogSourceConfigHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("catalog client not found"))
		return
	}

	sourceID := ps.ByName(CatalogSourceId)

	catalogNS := app.skillCatalogNamespace(namespace)
	deletedConfig, err := app.repositories.SkillCatalogSettingsRepository.DeleteSkillCatalogSourceConfig(ctx, client, catalogNS, sourceID)
	if err != nil {
		if errors.Is(err, repositories.ErrSkillCatalogCannotDeleteDefault) {
			app.forbiddenResponse(w, r, err.Error())
		} else if errors.Is(err, repositories.ErrSkillCatalogSourceNotFound) {
			app.notFoundResponse(w, r)
		} else if errors.Is(err, repositories.ErrSkillCatalogSourceConflict) {
			app.conflictResponse(w, r, err.Error())
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Best-effort cleanup: drop only this source's key from the shared
	// git-credentials Secret. Deleting the Secret itself would revoke every other
	// source's token and break the catalog pod's volume mount.
	//
	// A failure here must not fail the delete — the source is already gone from the
	// ConfigMap — but it leaves a revoked source's git token in the Secret, so it is
	// logged against the source id an operator can act on.
	if err := client.RemoveSecretKey(ctx, catalogNS, app.config.SkillCatalogGitCredentialsSecret, sourceID); err != nil {
		if sessionLogger, ok := ctx.Value(constants.TraceLoggerKey).(*slog.Logger); ok {
			sessionLogger.Warn("failed to remove git credentials for deleted skill catalog source",
				"sourceId", sourceID,
				"namespace", catalogNS,
				"secret", app.config.SkillCatalogGitCredentialsSecret,
				"error", err,
			)
		}
	}

	if err = app.WriteJSON(w, http.StatusOK, SkillCatalogSettingsSourceConfigEnvelope{Data: deletedConfig}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
