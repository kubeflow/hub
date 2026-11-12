package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

type ModelTransferJobListEnvelope Envelope[*models.ModelTransferJobList, None]
type ModelTransferJobEnvelope Envelope[*models.ModelTransferJob, None]
type ModelTransferJobEventsEnvelope Envelope[models.ModelTransferJobEventsResponse, None]

// GetAllModelTransferJobsHandler lists model transfer jobs.
//
//	@Summary	List All Model Transfer Jobs
//	@Tags		ModelRegistryService
//	@ID			getModelTransferJobs
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace			query		string	true	"Model registry namespace"
//	@Param		model_registry_id	path		string	true	"Model registry name"
//	@Param		jobNamespace		query		string	false	"Job namespace; omitted to list across namespaces"
//	@Success	200					{object}	ModelTransferJobListEnvelope
//	@Failure	400					{object}	ErrorEnvelope
//	@Failure	403					{object}	ErrorEnvelope
//	@Failure	500					{object}	ErrorEnvelope
//	@Router		/api/v1/model_registry/{model_registry_id}/model_transfer_jobs [get]
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
	if err := app.authorizeJobNamespace(ctx, client, jobNamespace); err != nil {
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

// GetModelTransferJobHandler returns a model transfer job.
//
//	@Summary	Get a Model Transfer Job
//	@Tags		ModelRegistryService
//	@ID			getModelTransferJob
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace			query		string	true	"Model registry namespace"
//	@Param		model_registry_id	path		string	true	"Model registry name"
//	@Param		job_name			path		string	true	"Transfer job name"
//	@Param		jobNamespace		query		string	true	"Namespace containing the job"
//	@Success	200					{object}	ModelTransferJobEnvelope
//	@Failure	400					{object}	ErrorEnvelope
//	@Failure	403					{object}	ErrorEnvelope
//	@Failure	404					{object}	ErrorEnvelope
//	@Failure	500					{object}	ErrorEnvelope
//	@Router		/api/v1/model_registry/{model_registry_id}/model_transfer_jobs/{job_name} [get]
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
	if err := app.authorizeJobNamespace(ctx, client, jobNamespace); err != nil {
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

// CreateModelTransferJobHandler creates a model transfer job.
//
//	@Summary	Create a Model Transfer Job
//	@Tags		ModelRegistryService
//	@ID			createModelTransferJob
//	@Accept		json
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace			query		string						true	"Model registry namespace"
//	@Param		model_registry_id	path		string						true	"Model registry name"
//	@Param		body				body		ModelTransferJobEnvelope	true	"Transfer job; data.namespace defaults to the registry namespace"
//	@Success	201					{object}	ModelTransferJobEnvelope
//	@Header		201					{string}	Location	"URL of the created transfer job"
//	@Failure	400					{object}	ErrorEnvelope
//	@Failure	403					{object}	ErrorEnvelope
//	@Failure	404					{object}	ErrorEnvelope
//	@Failure	500					{object}	ErrorEnvelope
//	@Router		/api/v1/model_registry/{model_registry_id}/model_transfer_jobs [post]
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

	if payload.Namespace == "" {
		payload.Namespace = namespace
	}
	if err := app.authorizeJobNamespace(ctx, client, payload.Namespace); err != nil {
		app.forbiddenResponse(w, r, err.Error())
		return
	}

	modelRegistryID := ps.ByName(ModelRegistryId)

	if modelRegistryID == "" {
		app.badRequestResponse(w, r, fmt.Errorf("model registry name is required"))
		return
	}

	newJob, err := app.repositories.ModelRegistry.CreateModelTransferJob(ctx, client, namespace, payload, modelRegistryID, app.config.DeploymentMode.IsFederatedMode(), app.podNamespace, app.config.BundlePaths)
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

// UpdateModelTransferJobHandler retries or replaces a model transfer job.
//
//	@Summary	Update a Model Transfer Job
//	@Tags		ModelRegistryService
//	@ID			updateModelTransferJob
//	@Accept		json
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace			query		string						true	"Model registry namespace"
//	@Param		model_registry_id	path		string						true	"Model registry name"
//	@Param		job_name			path		string						true	"Transfer job name"
//	@Param		deleteOldJob		query		boolean						false	"Delete the previous job"	default(false)
//	@Param		body				body		ModelTransferJobEnvelope	true	"Updated transfer job; data.namespace defaults to the registry namespace"
//	@Success	200					{object}	ModelTransferJobEnvelope
//	@Failure	400					{object}	ErrorEnvelope
//	@Failure	403					{object}	ErrorEnvelope
//	@Failure	404					{object}	ErrorEnvelope
//	@Failure	500					{object}	ErrorEnvelope
//	@Router		/api/v1/model_registry/{model_registry_id}/model_transfer_jobs/{job_name} [patch]
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

	if payload.Namespace == "" {
		payload.Namespace = namespace
	}
	if err := app.authorizeJobNamespace(ctx, client, payload.Namespace); err != nil {
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
		ctx, client, namespace, jobName, payload, deleteOldJob, modelRegistryID, app.config.DeploymentMode.IsFederatedMode(), app.podNamespace, app.config.BundlePaths)
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

// DeleteModelTransferJobHandler deletes a model transfer job.
//
//	@Summary		Delete a Model Transfer Job
//	@Description	Returns the deleted job with HTTP 200.
//	@Tags			ModelRegistryService
//	@ID				deleteModelTransferJob
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string	true	"Model registry namespace"
//	@Param			model_registry_id	path		string	true	"Model registry name"
//	@Param			job_name			path		string	true	"Transfer job name"
//	@Param			jobNamespace		query		string	true	"Namespace containing the job"
//	@Success		200					{object}	ModelTransferJobEnvelope
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id}/model_transfer_jobs/{job_name} [delete]
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
	if err := app.authorizeJobNamespace(ctx, client, jobNamespace); err != nil {
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

// GetModelTransferJobEventsHandler lists events for a model transfer job.
//
//	@Summary	Get Model Transfer Job Events
//	@Tags		ModelRegistryService
//	@ID			getModelTransferJobEvents
//	@Produce	json
//	@Security	KubeflowUserID
//	@Security	BearerAuth
//	@Param		namespace			query		string	true	"Model registry namespace"
//	@Param		model_registry_id	path		string	true	"Model registry name"
//	@Param		job_name			path		string	true	"Transfer job name"
//	@Param		jobNamespace		query		string	true	"Namespace containing the job"
//	@Success	200					{object}	ModelTransferJobEventsEnvelope
//	@Failure	400					{object}	ErrorEnvelope
//	@Failure	403					{object}	ErrorEnvelope
//	@Failure	404					{object}	ErrorEnvelope
//	@Failure	500					{object}	ErrorEnvelope
//	@Router		/api/v1/model_registry/{model_registry_id}/model_transfer_jobs/{job_name}/events [get]
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
	if err := app.authorizeJobNamespace(ctx, client, jobNamespace); err != nil {
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

// authorizeJobNamespace verifies the caller has service-level access in the
// namespace where the transfer-job handler will operate. The middleware
// (RequireAccessToMRService) authorizes the model-registry service namespace,
// but jobs legitimately run in a different namespace (the user's project), so
// a separate access check is required for that namespace. An empty namespace
// is left to the caller's existing handling (e.g. cluster-wide list).
func (app *App) authorizeJobNamespace(ctx context.Context, client kubernetes.KubernetesClientInterface, jobNamespace string) error {
	if jobNamespace == "" {
		return nil
	}
	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		return fmt.Errorf("missing request identity")
	}
	allowed, err := client.CanListServicesInNamespace(ctx, identity, jobNamespace)
	if err != nil {
		return fmt.Errorf("authorization check failed for namespace %q: %w", jobNamespace, err)
	}
	if !allowed {
		return fmt.Errorf("access denied to namespace %q", jobNamespace)
	}
	return nil
}
