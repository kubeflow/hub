package kubernetes

import (
	"context"
	"fmt"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mcpCatalogUpdateAttributes(namespace string) *authv1.ResourceAttributes {
	return &authv1.ResourceAttributes{
		Group:     "",
		Version:   "v1",
		Resource:  "configmaps",
		Verb:      "update",
		Namespace: namespace,
		Name:      McpCatalogSourceUserConfigMapName,
	}
}

func (kc *TokenKubernetesClient) CanUpdateMcpCatalogSources(
	ctx context.Context,
	_ *RequestIdentity,
	namespace string,
) (bool, error) {
	if namespace == "" {
		return false, fmt.Errorf("namespace is required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	review, err := kc.Client.AuthorizationV1().
		SelfSubjectAccessReviews().
		Create(ctx, &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: mcpCatalogUpdateAttributes(namespace),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("checking MCP source update permission: %w", err)
	}

	if review.Status.EvaluationError != "" {
		return false, fmt.Errorf(
			"evaluating MCP source update permission: %s",
			review.Status.EvaluationError,
		)
	}

	return review.Status.Allowed && !review.Status.Denied, nil
}

func (kc *InternalKubernetesClient) CanUpdateMcpCatalogSources(
	ctx context.Context,
	identity *RequestIdentity,
	namespace string,
) (bool, error) {
	if namespace == "" {
		return false, fmt.Errorf("namespace is required")
	}
	if identity == nil || identity.UserID == "" {
		return false, fmt.Errorf("request identity is required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	review, err := kc.Client.AuthorizationV1().
		SubjectAccessReviews().
		Create(ctx, &authv1.SubjectAccessReview{
			Spec: authv1.SubjectAccessReviewSpec{
				User:               identity.UserID,
				Groups:             identity.Groups,
				ResourceAttributes: mcpCatalogUpdateAttributes(namespace),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("checking MCP source update permission: %w", err)
	}

	if review.Status.EvaluationError != "" {
		return false, fmt.Errorf(
			"evaluating MCP source update permission: %s",
			review.Status.EvaluationError,
		)
	}

	return review.Status.Allowed && !review.Status.Denied, nil
}
