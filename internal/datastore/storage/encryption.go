package storage

import (
	"os"

	"github.com/padok-team/burrito/internal/burrito/config"
	"github.com/padok-team/burrito/internal/utils/encryption"
)

type EncryptionManager struct {
	defaultEncryptor    *encryption.Encryptor
	namespaceEncryptors map[string]*encryption.Encryptor
	config              config.EncryptionConfig
}

func NewEncryptionManager(config config.EncryptionConfig) *EncryptionManager {
	em := &EncryptionManager{
		namespaceEncryptors: make(map[string]*encryption.Encryptor),
		config:              config,
	}

	// Read encryption key from environment variable instead of config
	encryptionKey := os.Getenv("BURRITO_DATASTORE_STORAGE_ENCRYPTION_KEY")

	if config.Enabled && encryptionKey != "" {
		em.defaultEncryptor = encryption.NewEncryptor(encryptionKey)
	} else {
		em.defaultEncryptor = nil
		// // Create namespace-specific encryptors
		// for namespace, key := range config.NamespaceKeys {
		// 	em.namespaceEncryptors[namespace] = encryption.NewEncryptor(key)
		// }
	}

	return em
}

func (em *EncryptionManager) GetEncryptor(namespace string) *encryption.Encryptor {
	if encryptor, exists := em.namespaceEncryptors[namespace]; exists {
		return encryptor
	}

	return em.defaultEncryptor
}

// try to get the encryptor for the namespace, if not found, return the default encryptor
func (em *EncryptionManager) Encrypt(namespace string, plaintext []byte) ([]byte, error) {
	if em.defaultEncryptor == nil {
		return plaintext, nil
	}

	encryptor := em.GetEncryptor(namespace)
	if encryptor == nil {
		return plaintext, nil
	}

	return encryptor.Encrypt(plaintext)
}

func (em *EncryptionManager) Decrypt(namespace string, ciphertext []byte) ([]byte, error) {
	if em.defaultEncryptor == nil {
		return ciphertext, nil
	}

	encryptor := em.GetEncryptor(namespace)
	if encryptor == nil {
		return ciphertext, nil
	}

	// Try to decrypt the data. If it fails, return the original ciphertext as this might be a migration from an unencrypted state
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		return ciphertext, nil
	}

	return decrypted, nil
}
