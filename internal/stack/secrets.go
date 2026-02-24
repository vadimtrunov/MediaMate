package stack

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// defaultPasswordLen is the number of random bytes used for generated
// passwords. A value of 16 produces a 32-character hex string.
const defaultPasswordLen = 16

// GeneratePassword returns a cryptographically random password of the given
// byte length, hex-encoded (so the returned string is 2*n characters long).
// A length of 16 produces a 32-character hex string.
func GeneratePassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateSecrets produces a map of secret names to generated passwords for
// the internal services in the stack. These are used to populate the .env file.
func GenerateSecrets(cfg *Config) (map[string]string, error) {
	secrets := make(map[string]string)

	if cfg.HasComponent(ComponentQBittorrent) {
		p, err := GeneratePassword(defaultPasswordLen)
		if err != nil {
			return nil, err
		}
		secrets["QBITTORRENT_PASSWORD"] = p
	}

	if cfg.HasComponent(ComponentTransmission) {
		p, err := GeneratePassword(defaultPasswordLen)
		if err != nil {
			return nil, err
		}
		secrets["TRANSMISSION_PASSWORD"] = p
	}

	if cfg.HasComponent(ComponentDeluge) {
		p, err := GeneratePassword(defaultPasswordLen)
		if err != nil {
			return nil, err
		}
		secrets["DELUGE_PASSWORD"] = p
	}

	return secrets, nil
}
