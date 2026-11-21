package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
)

const CheckNamespaceRegistryAccessPath = ApiPathPrefix + "/check-namespace-registry-access"

type CheckNamespaceRegistryAccessRequest struct {
	Namespace         string `json:"namespace"`
	RegistryName      string `json:"registryName"`
	RegistryNamespace string `json:"registryNamespace"`
}

type CheckNamespaceRegistryAccessRequestEnvelope Envelope[CheckNamespaceRegistryAccessRequest, None]

type CheckNamespaceRegistryAccessResponse struct {
	HasAccess   bool `json:"hasAccess"`
	CannotCheck bool `json:"cannotCheck"`
}

type CheckNamespaceRegistryAccessEnvelope Envelope[CheckNamespaceRegistryAccessResponse, None]

// CheckNamespaceRegistryAccessHandler checks access between a namespace and registry.
//
//	@Summary		Check namespace access to a model registry
//	@Description	Checks whether a namespace can access the registry. cannotCheck is true when subject access reviews are forbidden.
//	@Tags			K8SOperation
//	@ID				checkNamespaceRegistryAccess
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			body	body		CheckNamespaceRegistryAccessRequestEnvelope	true	"Namespace and target registry"
//	@Success		200		{object}	CheckNamespaceRegistryAccessEnvelope
//	@Failure		400		{object}	ErrorEnvelope
//	@Failure		500		{object}	ErrorEnvelope
//	@Router			/api/v1/check-namespace-registry-access [post]
func (app *App) CheckNamespaceRegistryAccessHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	if err := app.kubernetesClientFactory.ValidateRequestIdentity(identity); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var envelope CheckNamespaceRegistryAccessRequestEnvelope
	if err := app.ReadJSON(w, r, &envelope); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	req := envelope.Data

	if req.Namespace == "" || req.RegistryName == "" || req.RegistryNamespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("namespace, registryName and registryNamespace are required"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	hasAccess, err := client.CanNamespaceAccessRegistry(ctx, identity, req.Namespace, req.RegistryName, req.RegistryNamespace)
	if err != nil {
		if errors.Is(err, kubernetes.ErrSARForbidden) {
			resp := CheckNamespaceRegistryAccessEnvelope{
				Data: CheckNamespaceRegistryAccessResponse{HasAccess: false, CannotCheck: true},
			}
			if writeErr := app.WriteJSON(w, http.StatusOK, resp, nil); writeErr != nil {
				app.serverErrorResponse(w, r, writeErr)
			}
			return
		}
		app.serverErrorResponse(w, r, fmt.Errorf("namespace registry access check failed: %w", err))
		return
	}

	resp := CheckNamespaceRegistryAccessEnvelope{
		Data: CheckNamespaceRegistryAccessResponse{HasAccess: hasAccess},
	}
	if err := app.WriteJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
