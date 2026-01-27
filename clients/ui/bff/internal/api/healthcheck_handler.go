package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	// imported for swag documentation
	_ "github.com/kubeflow/model-registry/ui/bff/internal/models"
)

// HealthcheckHandler returns the health status of the application.
//
//	@Summary		Health check
//	@Description	Returns the health status of the application including version info
//	@Tags			healthcheck
//	@ID				healthcheck
//	@Produce		json
//	@Success		200	{object}	models.HealthCheckModel	"Successful healthcheck response"
//	@Failure		500	{object}	ErrorEnvelope			"Internal server error"
//	@Router			/healthcheck [get]
func (app *App) HealthcheckHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
