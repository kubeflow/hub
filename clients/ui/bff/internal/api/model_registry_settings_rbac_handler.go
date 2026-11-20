package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	helper "github.com/kubeflow/hub/ui/bff/internal/helpers"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/hub/ui/bff/internal/repositories"
)

type CertificateListEnvelope Envelope[models.CertificateList, None]
type RoleBindingListEnvelope Envelope[models.RoleBindingList, None]
type RoleBindingEnvelope Envelope[models.RoleBinding, None]

const (
	RoleBindingNameParam = "roleBindingName"
)

// GetCertificatesHandler handles GET /api/v1/settings/certificates
// STUB IMPLEMENTATION — pending definition of how certificates are stored.
func (app *App) GetCertificatesHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// TODO: Implement actual logic: use K8s client to list Secrets and ConfigMaps in relevant namespaces
	// For now, return empty data rather than hardcoded fake data.
	result := CertificateListEnvelope{
		Metadata: nil,
		Data:     models.CertificateList{},
	}

	err := app.WriteJSON(w, http.StatusOK, result, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetRoleBindingsHandler handles GET /api/v1/settings/role_bindings
// Returns real RoleBindings from Kubernetes, filtered by app.kubernetes.io/part-of=model-registry.
func (app *App) GetRoleBindingsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	ctxLogger := helper.GetContextLoggerFromReq(r)

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		ctxLogger.Error("failed to get kubernetes client", "error", err)
		app.serverErrorResponse(w, r, errors.New("kubernetes client not available"))
		return
	}

	bindingList, err := app.repositories.ModelRegistryRBAC.GetRoleBindings(ctx, client, namespace)
	if err != nil {
		ctxLogger.Error("failed to list role bindings", "namespace", namespace, "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	result := RoleBindingListEnvelope{
		Metadata: nil,
		Data:     bindingList,
	}

	if err := app.WriteJSON(w, http.StatusOK, result, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// CreateRoleBindingHandler handles POST /api/v1/settings/role_bindings
// Creates a new Kubernetes RoleBinding for a model-registry.
func (app *App) CreateRoleBindingHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	ctxLogger := helper.GetContextLoggerFromReq(r)

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	var input RoleBindingEnvelope
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Data.Name == "" {
		app.badRequestResponse(w, r, fmt.Errorf("role binding name is required"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		ctxLogger.Error("failed to get kubernetes client", "error", err)
		app.serverErrorResponse(w, r, errors.New("kubernetes client not available"))
		return
	}

	created, err := app.repositories.ModelRegistryRBAC.CreateRoleBinding(ctx, client, namespace, &input.Data)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleBindingNameEmpty) {
			app.badRequestResponse(w, r, err)
			return
		}
		ctxLogger.Error("failed to create role binding", "namespace", namespace, "name", input.Data.Name, "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	result := RoleBindingEnvelope{
		Metadata: nil,
		Data:     *created,
	}

	w.Header().Set("Location", r.URL.JoinPath(created.Name).String())
	if err := app.WriteJSON(w, http.StatusCreated, result, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// DeleteRoleBindingHandler handles DELETE /api/v1/settings/role_bindings/{roleBindingName}
// Deletes a Kubernetes RoleBinding by name.
func (app *App) DeleteRoleBindingHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	ctxLogger := helper.GetContextLoggerFromReq(r)

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	roleBindingName := ps.ByName(RoleBindingNameParam)
	if roleBindingName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("role binding name is required"))
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		ctxLogger.Error("failed to get kubernetes client", "error", err)
		app.serverErrorResponse(w, r, errors.New("kubernetes client not available"))
		return
	}

	err = app.repositories.ModelRegistryRBAC.DeleteRoleBinding(ctx, client, namespace, roleBindingName)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleBindingNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		ctxLogger.Error("failed to delete role binding", "namespace", namespace, "name", roleBindingName, "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PatchRoleBindingHandler handles PATCH /api/v1/settings/role_bindings/{roleBindingName}
// Updates the subjects of an existing Kubernetes RoleBinding.
func (app *App) PatchRoleBindingHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	ctxLogger := helper.GetContextLoggerFromReq(r)

	namespace, ok := ctx.Value(constants.NamespaceHeaderParameterKey).(string)
	if !ok || namespace == "" {
		app.badRequestResponse(w, r, fmt.Errorf("missing namespace in context"))
		return
	}

	roleBindingName := ps.ByName(RoleBindingNameParam)
	if roleBindingName == "" {
		app.badRequestResponse(w, r, fmt.Errorf("role binding name is required"))
		return
	}

	var input RoleBindingEnvelope
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	client, err := app.kubernetesClientFactory.GetClient(ctx)
	if err != nil {
		ctxLogger.Error("failed to get kubernetes client", "error", err)
		app.serverErrorResponse(w, r, errors.New("kubernetes client not available"))
		return
	}

	patched, err := app.repositories.ModelRegistryRBAC.PatchRoleBinding(ctx, client, namespace, roleBindingName, &input.Data)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleBindingNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		if errors.Is(err, repositories.ErrRoleBindingNameEmpty) {
			app.badRequestResponse(w, r, err)
			return
		}
		ctxLogger.Error("failed to patch role binding", "namespace", namespace, "name", roleBindingName, "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	result := RoleBindingEnvelope{
		Metadata: nil,
		Data:     *patched,
	}

	if err := app.WriteJSON(w, http.StatusOK, result, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
