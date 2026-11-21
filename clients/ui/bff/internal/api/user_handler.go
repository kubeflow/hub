package api

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type UserEnvelope Envelope[*models.User, None]

// UserHandler returns configuration for the current user.
//
//	@Summary		Get configuration info
//	@Description	Gets the user and deployment configuration needed by the UI.
//	@Tags			K8SOperation
//	@ID				getConfig
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Success		200	{object}	UserEnvelope
//	@Failure		400	{object}	ErrorEnvelope
//	@Failure		500	{object}	ErrorEnvelope
//	@Router			/api/v1/user [get]
func (app *App) UserHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	ctx := r.Context()
	identity, ok := ctx.Value(constants.RequestIdentityKey).(*kubernetes.RequestIdentity)
	if !ok || identity == nil {
		app.badRequestResponse(w, r, fmt.Errorf("missing RequestIdentity in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to get Kubernetes client: %w", err))
		return
	}

	user, err := app.repositories.User.GetUser(client, identity)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	userRes := UserEnvelope{
		Data: user,
	}

	err = app.WriteJSON(w, http.StatusOK, userRes, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
