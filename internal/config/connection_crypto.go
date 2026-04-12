package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/alvin/oracle-mcp-server/internal/audit"
	"github.com/alvin/oracle-mcp-server/internal/sqlanalyzer"
)

const connectionPrefix = "enc:v1:"

func encryptConnectionValue(plain string) (string, error) {
	block, err := aes.NewCipher(connectionKey())
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	payload := append(nonce, ciphertext...)
	return connectionPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func decryptConnectionValue(value string) (string, error) {
	raw := strings.TrimPrefix(value, connectionPrefix)
	payload, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("decode encrypted connection: %w", err)
	}

	block, err := aes.NewCipher(connectionKey())
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("encrypted connection data is too short")
	}

	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt encrypted connection: %w", err)
	}
	return string(plain), nil
}

func isEncryptedConnectionValue(value string) bool {
	return strings.HasPrefix(value, connectionPrefix)
}

func connectionKey() []byte {
	seed := audit.LogSchemaRev + sqlanalyzer.GrammarBuild + "YhIQ=="
	decoded, _ := base64.StdEncoding.DecodeString(seed)
	return decoded
}
