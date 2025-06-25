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
	var allRepositories []configv1alpha1.TerraformRepository

	// Iterate over all configured namespaces instead of cluster-wide listing
	for _, namespace := range a.config.Controller.Namespaces {
		repositories := &configv1alpha1.TerraformRepositoryList{}
		err := a.Client.List(context.Background(), repositories, client.InNamespace(namespace))
		if err != nil {
			log.Errorf("could not list TerraformRepositories in namespace %s: %s", namespace, err)
			continue
		}
		allRepositories = append(allRepositories, repositories.Items...)
	}

	results := []repository{}
	for _, r := range allRepositories {
		results = append(results, repository{
			Name: fmt.Sprintf("%s/%s", r.Namespace, r.Name),
		})
	}
	return c.JSON(http.StatusOK, &repositoriesResponse{
		Results: results,
	},
	)
}
