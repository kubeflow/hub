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
//	@Tags			healthcheck
//	@ID				getHealthcheck
//	@Produce		application/json
//	@Success		200	{object}	healthcheck.HealthCheckModel	"Successful healthcheck response"
//	@Failure		401	{object}	ErrorEnvelope					"Unauthorized. Authentication is required."
//	@Failure		403	{object}	ErrorEnvelope					"Forbidden. User does not have permission to access the resource."
//	@Failure		404	{object}	ErrorEnvelope					"Not Found. Resource does not exist."
//	@Failure		422	{object}	ErrorEnvelope					"Unprocessable Entity. Validation error."
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
