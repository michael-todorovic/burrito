package api

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/padok-team/burrito/internal/burrito/config"
	datastore "github.com/padok-team/burrito/internal/datastore/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type API struct {
	config     *config.Config
	Client     client.Client
	Datastore  datastore.Client
	namespace  string
}

func New(c *config.Config) *API {
	return &API{
		config:    c,
		namespace: getCurrentNamespace(), // Keep for backward compatibility
	}
}

// getNamespaces returns the list of tenant namespaces to operate on
func (a *API) getNamespaces() []string {
	// Use tenant namespaces from controller config if available, otherwise fall back to current namespace
	namespaces := a.config.Controller.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{a.namespace}
	}
	return namespaces
}

// getCurrentNamespace tries to determine the current namespace from environment variables or service account
func getCurrentNamespace() string {
	// First, try to get namespace from environment variable (commonly set in k8s deployments)
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	// Second, try to read from service account namespace file
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); ns != "" {
			return ns
		}
	}

	// Third, try the NAMESPACE environment variable
	if ns := os.Getenv("NAMESPACE"); ns != "" {
		return ns
	}

	// Default fallback
	return "burrito"
}
