package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

type ModelTransferJobListEnvelope Envelope[*models.ModelTransferJobList, None]
type ModelTransferJobEnvelope Envelope[*models.ModelTransferJob, None]
type ModelTransferJobEventsEnvelope Envelope[models.ModelTransferJobEventsResponse, None]

func (app *App) GetAllModelTransferJobsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("kubernetes client not found"))
		return
	}

	modelRegistryID := ps.ByName(ModelRegistryId)
	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}

	jobNamespace := r.URL.Query().Get("jobNamespace")
	if err := ensureNamespaceMatch(namespace, jobNamespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}

	transferJobs, err := app.repositories.ModelRegistry.GetAllModelTransferJobs(ctx, client, namespace, modelRegistryID, jobNamespace)
	if err != nil {
		if errors.Is(err, repositories.ErrForbidden) {
			app.forbiddenResponse(w, r, err.Error())
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	response := ModelTransferJobListEnvelope{
		Data: transferJobs,
	}

	err = app.WriteJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetModelTransferJobHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("kubernetes client not found"))
		return
	}

	jobName := ps.ByName(ModelTransferJobName)
	if jobName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("job name is required"))
		return
	}

	jobNamespace, err := getRequiredJobNamespace(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := ensureNamespaceMatch(namespace, jobNamespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}
	modelRegistryID := ps.ByName(ModelRegistryId)
	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}

	modelTransferJob, err := app.repositories.ModelRegistry.GetModelTransferJob(ctx, client, jobNamespace, jobName, modelRegistryID)

	if err != nil {
		if errors.Is(err, repositories.ErrJobNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		if errors.Is(err, repositories.ErrJobValidationFailed) {
			app.badRequestResponse(w, r, err)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}
	response := ModelTransferJobEnvelope{Data: modelTransferJob}

	err = app.WriteJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *App) CreateModelTransferJobHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("kubernetes client not found"))
		return
	}

	var envelope ModelTransferJobEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %w", err))
		return
	}

	if envelope.Data == nil {
		app.badRequestResponse(w, r, fmt.Errorf("data is required"))
		return
	}

	payload := *envelope.Data

	// The transfer-job resources are created in payload.Namespace, so it must
	// stay inside the namespace authorized by RequireAccessToMRService. Default
	// an unset namespace to the authorized one and reject any other.
	if payload.Namespace == "" {
		payload.Namespace = namespace
	}
	if err := ensureNamespaceMatch(namespace, payload.Namespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}

	modelRegistryID := ps.ByName(ModelRegistryId)

	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}

	newJob, err := app.repositories.ModelRegistry.CreateModelTransferJob(ctx, client, namespace, payload, modelRegistryID, app.config.DeploymentMode.IsFederatedMode(), app.podNamespace)
	if err != nil {
		if errors.Is(err, repositories.ErrJobValidationFailed) {
			app.badRequestResponse(w, r, err)
			return
		}
		if errors.Is(err, repositories.ErrModelRegistryNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	modelTransferJob := ModelTransferJobEnvelope{Data: newJob}

	w.Header().Set("Location", r.URL.JoinPath(modelTransferJob.Data.Name).String())
	writeErr := app.WriteJSON(w, http.StatusCreated, modelTransferJob, nil)
	if writeErr != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("error writing JSON: %w", writeErr))
		return
	}
}

func (app *App) UpdateModelTransferJobHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("kubernetes client not found"))
		return
	}

	var envelope ModelTransferJobEnvelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("error decoding JSON: %w", err))
		return
	}
	if envelope.Data == nil {
		app.badRequestResponse(w, r, fmt.Errorf("data is required"))
		return
	}
	payload := *envelope.Data

	// Keep the update inside the authorized namespace: the destination Secret and
	// Job the repository reads and writes use payload.Namespace.
	if payload.Namespace == "" {
		payload.Namespace = namespace
	}
	if err := ensureNamespaceMatch(namespace, payload.Namespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}

	jobName := ps.ByName(ModelTransferJobName)
	if jobName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("job name is required"))
		return
	}
	modelRegistryID := ps.ByName(ModelRegistryId)
	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}
	deleteOldJob := r.URL.Query().Get("deleteOldJob") == "true"

	updatedJob, err := app.repositories.ModelRegistry.UpdateModelTransferJob(
		ctx, client, namespace, jobName, payload, deleteOldJob, modelRegistryID, app.config.DeploymentMode.IsFederatedMode(), app.podNamespace)
	if err != nil {
		if errors.Is(err, repositories.ErrJobNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		if errors.Is(err, repositories.ErrJobValidationFailed) {
			app.badRequestResponse(w, r, err)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	modelTransferJob := ModelTransferJobEnvelope{
		Data: updatedJob,
	}

	err = app.WriteJSON(w, http.StatusOK, modelTransferJob, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *App) DeleteModelTransferJobHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("kubernetes client not found"))
		return
	}

	jobName := ps.ByName(ModelTransferJobName)
	if jobName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("job name is required"))
		return
	}

	jobNamespace, err := getRequiredJobNamespace(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := ensureNamespaceMatch(namespace, jobNamespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}

	modelRegistryID := ps.ByName(ModelRegistryId)
	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}

	deletedJob, err := app.repositories.ModelRegistry.DeleteModelTransferJob(ctx, client, jobNamespace, jobName, modelRegistryID)
	if err != nil {
		if errors.Is(err, repositories.ErrJobNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	response := ModelTransferJobEnvelope{Data: deletedJob}

	err = app.WriteJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *App) GetModelTransferJobEventsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, errors.New("kubernetes client not found"))
		return
	}

	jobName := ps.ByName(ModelTransferJobName)
	if jobName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("job name is required"))
		return
	}

	modelRegistryID := ps.ByName(ModelRegistryId)
	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}

	jobNamespace, err := getRequiredJobNamespace(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := ensureNamespaceMatch(namespace, jobNamespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}

	events, err := app.repositories.ModelRegistry.GetModelTransferJobEvents(ctx, client, jobNamespace, jobName, modelRegistryID)
	if err != nil {
		if errors.Is(err, repositories.ErrJobNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	response := ModelTransferJobEventsEnvelope{
		Data: models.ModelTransferJobEventsResponse{Events: events},
	}

	err = app.WriteJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func getRequiredJobNamespace(r *http.Request) (string, error) {
	jobNamespace := r.URL.Query().Get("jobNamespace")
	if jobNamespace == "" {
		return "", fmt.Errorf("missing required query parameter: jobNamespace")
	}
	return jobNamespace, nil
}

// ensureNamespaceMatch closes the model-transfer-job BOLA. RequireAccessToMRService
// runs the SubjectAccessReview against the `namespace` query parameter, but the
// handlers operate on a namespace taken from `jobNamespace` (read/list/events/delete)
// or the request body (create/update). If that operated namespace differs from the
// authorized one, the caller could act on another tenant's namespace. Requiring the
// two to match keeps every operation inside the namespace the caller was authorized
// for. An empty operated namespace is left to the caller's existing handling.
func ensureNamespaceMatch(authorized, operated string) error {
	if operated != "" && operated != authorized {
		return fmt.Errorf("namespace %q does not match the authorized namespace %q", operated, authorized)
	}
	return nil
}
