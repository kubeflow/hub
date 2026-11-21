/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/openapi"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// GetSwaggerHandler serves the Swagger UI for interactive API documentation.
func (app *App) GetSwaggerHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// http-swagger uses the Swag v1 registry. Serve the embedded v2-generated
	// document directly so the UI and on-disk OpenAPI 3.1 spec stay identical.
	if r.URL.Path == SwaggerDocPath {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(openapi.Document); err != nil {
			app.LogError(r, err)
		}
		return
	}

	httpSwagger.Handler(
		httpSwagger.URL(SwaggerDocPath),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	).ServeHTTP(w, r)
}
