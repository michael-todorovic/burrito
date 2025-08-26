package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	configv1alpha1 "github.com/padok-team/burrito/api/v1alpha1"
	"github.com/padok-team/burrito/internal/annotations"
	"github.com/padok-team/burrito/internal/server/utils"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type layer struct {
	UID              string                 `json:"uid"`
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace"`
	Repository       string                 `json:"repository"`
	Branch           string                 `json:"branch"`
	Path             string                 `json:"path"`
	State            string                 `json:"state"`
	RunCount         int                    `json:"runCount"`
	LastRun          Run                    `json:"lastRun"`
	LastRunAt        string                 `json:"lastRunAt"`
	LastResult       string                 `json:"lastResult"`
	IsRunning        bool                   `json:"isRunning"`
	IsPR             bool                   `json:"isPR"`
	LatestRuns       []Run                  `json:"latestRuns"`
	ManualSyncStatus utils.ManualSyncStatus `json:"manualSyncStatus"`
}

type Run struct {
	Name   string `json:"id"`
	Commit string `json:"commit"`
	Date   string `json:"date"`
	Action string `json:"action"`
}

type layersResponse struct {
	Results []layer `json:"results"`
}

func (a *API) getLayersAndRuns() ([]configv1alpha1.TerraformLayer, map[string]configv1alpha1.TerraformRun, error) {
	allLayers := []configv1alpha1.TerraformLayer{}
	indexedRuns := map[string]configv1alpha1.TerraformRun{}
	
	// Get current tenant namespaces dynamically
	namespaces := a.getNamespaces()
	
	// Collect layers and runs from all configured tenant namespaces
	for _, namespace := range namespaces {
		layers := &configv1alpha1.TerraformLayerList{}
		err := a.Client.List(context.Background(), layers, &client.ListOptions{
			Namespace: namespace,
		})
		if err != nil {
			log.Errorf("could not list TerraformLayers in namespace %s: %s", namespace, err)
			continue // Continue with other namespaces even if one fails
		}
		allLayers = append(allLayers, layers.Items...)
		
		runs := &configv1alpha1.TerraformRunList{}
		err = a.Client.List(context.Background(), runs, &client.ListOptions{
			Namespace: namespace,
		})
		if err != nil {
			log.Errorf("could not list TerraformRuns in namespace %s: %s", namespace, err)
			continue // Continue with other namespaces even if one fails
		}
		
		for _, run := range runs.Items {
			indexedRuns[fmt.Sprintf("%s/%s", run.Namespace, run.Name)] = run
		}
	}
	
	return allLayers, indexedRuns, nil
}

func (a *API) LayersHandler(c echo.Context) error {
	layers, runs, err := a.getLayersAndRuns()
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("could not list terraform layers or runs: %s", err))
	}
	results := []layer{}
	for _, l := range layers {
		if err != nil {
			log.Errorf("could not get latest run for layer %s: %s", l.Name, err)
		}
		run, ok := runs[fmt.Sprintf("%s/%s", l.Namespace, l.Status.LastRun.Name)]
		runAPI := Run{}
		running := false
		if ok {
			runAPI = Run{
				Name:   run.Name,
				Commit: "",
				Date:   run.CreationTimestamp.Format(time.RFC3339),
				Action: run.Spec.Action,
			}
			running = runStillRunning(run)
		}
		results = append(results, layer{
			UID:              string(l.UID),
			Name:             l.Name,
			Namespace:        l.Namespace,
			Repository:       fmt.Sprintf("%s/%s", l.Spec.Repository.Namespace, l.Spec.Repository.Name),
			Branch:           l.Spec.Branch,
			Path:             l.Spec.Path,
			State:            a.getLayerState(l),
			RunCount:         len(l.Status.LatestRuns),
			LastRun:          runAPI,
			LastRunAt:        l.Status.LastRun.Date.Format(time.RFC3339),
			LastResult:       l.Status.LastResult,
			IsRunning:        running,
			IsPR:             a.isLayerPR(l),
			LatestRuns:       transformLatestRuns(l.Status.LatestRuns),
			ManualSyncStatus: utils.GetManualSyncStatus(l),
		})
	}
	return c.JSON(http.StatusOK, &layersResponse{
		Results: results,
	},
	)
}

func runStillRunning(run configv1alpha1.TerraformRun) bool {
	if run.Status.State != "Failed" && run.Status.State != "Succeeded" {
		return true
	}
	return false
}

func transformLatestRuns(runs []configv1alpha1.TerraformLayerRun) []Run {
	results := []Run{}
	for _, r := range runs {
		results = append(results, Run{
			Name:   r.Name,
			Commit: r.Commit,
			Date:   r.Date.Format(time.RFC3339),
			Action: r.Action,
		})
	}
	return results
}

func (a *API) getLayerState(layer configv1alpha1.TerraformLayer) string {
	state := "success"
	switch {
	case len(layer.Status.Conditions) == 0:
		state = "disabled"
	case layer.Status.State == "ApplyNeeded":
		if layer.Status.LastResult == "Plan: 0 to create, 0 to update, 0 to delete" {
			state = "success"
		} else {
			state = "warning"
		}
	case layer.Status.State == "PlanNeeded":
		state = "warning"
	}
	if layer.Annotations[annotations.LastPlanSum] == "" {
		state = "error"
	}
	if layer.Annotations[annotations.LastApplySum] != "" && layer.Annotations[annotations.LastApplySum] == "" {
		state = "error"
	}
	return state
}

func (a *API) isLayerPR(layer configv1alpha1.TerraformLayer) bool {
	if len(layer.OwnerReferences) == 0 {
		return false
	}
	return layer.OwnerReferences[0].Kind == "TerraformPullRequest"
}
