package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	// imported for swag documentation
	_ "github.com/kubeflow/hub/ui/bff/internal/models/healthcheck"
)

// HealthcheckHandler returns the health status of the application.
//
//	@Summary		Returns the health status of the application
//	@Description	Provides a healthcheck response indicating the status of key services.
//	@Tags			K8SOperation
//	@ID				healthcheck
//	@Produce		application/json
//	@Success		200	{object}	healthcheck.HealthCheckModel	"Successful healthcheck response"
//	@Failure		500	{object}	ErrorEnvelope					"Internal server error"
//	@Router			/healthcheck [get]
func (app *App) HealthcheckHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	healthCheck, err := app.repositories.HealthCheck.HealthCheck(Version)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, http.StatusOK, healthCheck, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
