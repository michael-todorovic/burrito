package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/padok-team/burrito/internal/burrito/config"
	"github.com/padok-team/burrito/internal/datastore/api"
	"github.com/padok-team/burrito/internal/datastore/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteWithEncryption(t *testing.T) {
	// Set up encryption key environment variable
	encryptionKey := "test-encryption-key-for-api-testing-123"
	err := os.Setenv("BURRITO_DATASTORE_STORAGE_ENCRYPTION_KEY", encryptionKey)
	require.NoError(t, err)
	defer os.Unsetenv("BURRITO_DATASTORE_STORAGE_ENCRYPTION_KEY")

	// Create storage with encryption enabled
	testConfig := config.Config{
		Datastore: config.DatastoreConfig{
			Storage: config.StorageConfig{
				Mock: true,
				Encryption: config.EncryptionConfig{
					Enabled: true,
				},
			},
		},
	}

	encryptedStorage := storage.New(testConfig)
	encryptedAPI := &api.API{}
	encryptedAPI.Storage = encryptedStorage

	// Test data
	body := []byte(`{"format_version":"1.1","terraform_version":"1.0.0","planned_values":{}}`)

	// Create echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/plans?namespace=encrypted-test&layer=test-layer&run=test-run&attempt=0&format=json", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	context := e.NewContext(req, rec)

	// Store plan with encryption
	err = encryptedAPI.PutPlanHandler(context)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify that data was stored encrypted by checking the raw backend
	// The encrypted data should be different from the original
	storedData, err := encryptedStorage.Backend.Get("layers/encrypted-test/test-layer/test-run/0/plan.json")
	require.NoError(t, err)
	assert.NotEqual(t, body, storedData, "stored data should be encrypted and different from original")

	// Verify that the storage layer can decrypt it correctly
	retrievedData, err := encryptedStorage.GetPlan("encrypted-test", "test-layer", "test-run", "0", "json")
	require.NoError(t, err)
	assert.Equal(t, body, retrievedData, "decrypted data should match original")
}
