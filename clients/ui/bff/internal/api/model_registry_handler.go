package api

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type ModelRegistryListEnvelope Envelope[[]models.ModelRegistryModel, None]
type ModelRegistryEnvelope Envelope[models.ModelRegistryModel, None]

// GetAllModelRegistriesHandler lists accessible model registries.
//
//	@Summary		List All Model Registries
//	@Description	Gets model registries the current user can access in the namespace.
//	@Tags			K8SOperation
//	@ID				getModelRegistries
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Model registry namespace"
//	@Success		200			{object}	ModelRegistryListEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		403			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry [get]
func (app *App) GetAllModelRegistriesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	namespace, ok := r.Context().Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in the context"))
	}

	client, err := app.kubernetesClientFactory.GetClient(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	registries, err := app.repositories.ModelRegistry.GetAllModelRegistriesWithMode(r.Context(), client, namespace, app.config.DeploymentMode.IsFederatedMode())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	modelRegistryRes := ModelRegistryListEnvelope{
		Data: registries,
	}

	err = app.WriteJSON(w, http.StatusOK, modelRegistryRes, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
