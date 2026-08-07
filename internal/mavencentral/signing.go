package mavencentral

import (
	"bytes"
	"context"
	"crypto"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

const UbuntuKeyserver = "https://keyserver.ubuntu.com"

// OpenPGPSigner signs deterministically with one persistent publisher key.
type OpenPGPSigner struct {
	entity *openpgp.Entity
	epoch  time.Time
}

// SigningKey describes a locally persisted signing identity.
type SigningKey struct {
	Signer      *OpenPGPSigner
	Path        string
	Fingerprint string
	PublicArmor []byte
	Created     bool
}

// EnsureSigningKey loads or creates the publisher's persistent OpenPGP key.
// Generation happens once; subsequent release signatures are reproducible.
func EnsureSigningKey(path, name, email string, epoch time.Time) (SigningKey, error) {
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return SigningKey{}, err
		}
		path = filepath.Join(configDir, "goplus", "maven-central-private.asc")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return SigningKey{}, err
	}
	var entity *openpgp.Entity
	created := false
	if data, readErr := os.ReadFile(path); readErr == nil {
		entities, parseErr := openpgp.ReadArmoredKeyRing(bytes.NewReader(data))
		if parseErr != nil || len(entities) != 1 || entities[0].PrivateKey == nil {
			return SigningKey{}, fmt.Errorf("invalid Maven Central private key %s", path)
		}
		entity = entities[0]
	} else if !os.IsNotExist(readErr) {
		return SigningKey{}, readErr
	} else {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(email) == "" {
			return SigningKey{}, fmt.Errorf("developer name and email are required to create a signing key")
		}
		config := signingConfig(epoch)
		config.RSABits = 3072
		entity, err = openpgp.NewEntity(name, "GoForge Maven Central", email, config)
		if err != nil {
			return SigningKey{}, err
		}
		var private bytes.Buffer
		armored, err := armor.Encode(&private, openpgp.PrivateKeyType, nil)
		if err != nil {
			return SigningKey{}, err
		}
		if err := entity.SerializePrivate(armored, config); err != nil {
			return SigningKey{}, err
		}
		if err := armored.Close(); err != nil {
			return SigningKey{}, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return SigningKey{}, err
		}
		temporary := path + ".tmp"
		if err := os.WriteFile(temporary, private.Bytes(), 0o600); err != nil {
			return SigningKey{}, err
		}
		if err := os.Rename(temporary, path); err != nil {
			return SigningKey{}, err
		}
		created = true
	}
	publicArmor, err := serializePublic(entity)
	if err != nil {
		return SigningKey{}, err
	}
	return SigningKey{Signer: &OpenPGPSigner{entity: entity, epoch: epoch}, Path: path, Fingerprint: strings.ToUpper(hex.EncodeToString(entity.PrimaryKey.Fingerprint)), PublicArmor: publicArmor, Created: created}, nil
}

func (s *OpenPGPSigner) Sign(_ context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var signature bytes.Buffer
	if err := openpgp.ArmoredDetachSign(&signature, s.entity, file, signingConfig(s.epoch)); err != nil {
		return nil, err
	}
	return signature.Bytes(), nil
}

func (s *OpenPGPSigner) Verify(data, signature []byte) error {
	_, err := openpgp.CheckArmoredDetachedSignature(openpgp.EntityList{s.entity}, bytes.NewReader(data), bytes.NewReader(signature), signingConfig(s.epoch))
	return err
}

func signingConfig(epoch time.Time) *packet.Config {
	deterministic := false
	return &packet.Config{DefaultHash: crypto.SHA256, Time: func() time.Time { return epoch.UTC() }, NonDeterministicSignaturesViaNotation: &deterministic}
}

func serializePublic(entity *openpgp.Entity) ([]byte, error) {
	var output bytes.Buffer
	w, err := armor.Encode(&output, openpgp.PublicKeyType, nil)
	if err != nil {
		return nil, err
	}
	if err := entity.Serialize(w); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// PublishPublicKey idempotently submits the public identity to Ubuntu's HKP
// service, one of the keyservers explicitly supported by Maven Central.
func PublishPublicKey(ctx context.Context, client *http.Client, base, fingerprint string, publicArmor []byte) error {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if base == "" {
		base = UbuntuKeyserver
	}
	form := url.Values{"keytext": {string(publicArmor)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/pks/add", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 500 {
		return fmt.Errorf("publishing OpenPGP key: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	lookup := strings.TrimRight(base, "/") + "/pks/lookup?op=get&search=" + url.QueryEscape("0x"+fingerprint)
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, lookup, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil {
			data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			response.Body.Close()
			if response.StatusCode == http.StatusOK && bytes.Contains(data, []byte(openpgp.PublicKeyType)) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("OpenPGP key %s was not readable from the keyserver: %w", fingerprint, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}
