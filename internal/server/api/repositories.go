package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	configv1alpha1 "github.com/padok-team/burrito/api/v1alpha1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type repository struct {
	Name string `json:"name"`
}

type repositoriesResponse struct {
	Results []repository `json:"results"`
}

func (a *API) RepositoriesHandler(c echo.Context) error {
	allResults := []repository{}
	
	// Get current tenant namespaces dynamically
	namespaces := a.getNamespaces()
	
	// List repositories from all configured tenant namespaces
	for _, namespace := range namespaces {
		repositories := &configv1alpha1.TerraformRepositoryList{}
		err := a.Client.List(context.Background(), repositories, &client.ListOptions{
			Namespace: namespace,
		})
		if err != nil {
			log.Errorf("could not list TerraformRepositories in namespace %s: %s", namespace, err)
			continue // Continue with other namespaces even if one fails
		}

		for _, r := range repositories.Items {
			allResults = append(allResults, repository{
				Name: fmt.Sprintf("%s/%s", r.Namespace, r.Name),
			})
		}
	}

	return c.JSON(http.StatusOK, &repositoriesResponse{
		Results: allResults,
	})
}
