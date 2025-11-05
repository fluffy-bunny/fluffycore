package keymaterial

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	linq "github.com/ahmetb/go-linq/v3"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
)

type (
	service struct {
		keyMaterial   *fluffycore_contracts_jwtminter.KeyMaterial
		lock          *sync.RWMutex
		nextFetchTime time.Time
		signingKey    *fluffycore_contracts_jwtminter.SigningKey
		jwks          []*fluffycore_contracts_jwtminter.PublicJwk
	}
)

var stemService = (*service)(nil)

var _ fluffycore_contracts_jwtminter.IKeyMaterial = stemService

func (s *service) Ctor(keyMaterial *fluffycore_contracts_jwtminter.KeyMaterial) (fluffycore_contracts_jwtminter.IKeyMaterial, error) {
	return &service{
		keyMaterial: keyMaterial,
		lock:        &sync.RWMutex{},
	}, nil
}

// AddSingletonIKeyMaterial ...
func AddSingletonIKeyMaterial(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_jwtminter.IKeyMaterial](builder, stemService.Ctor)
}

func (s *service) _reloadKeys() {
	now := time.Now()
	if now.After(s.nextFetchTime) {
		//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
		s.lock.Lock()
		defer s.lock.Unlock()
		//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
		var signingKeys []*fluffycore_contracts_jwtminter.SigningKey
		linq.From(s.keyMaterial.SigningKeys).Where(func(c interface{}) bool {
			signingKey := c.(*fluffycore_contracts_jwtminter.SigningKey)
			if now.After(signingKey.NotBefore) && now.Before(signingKey.NotAfter) {
				return true
			}
			return false
		}).Select(func(c interface{}) interface{} {
			signingKey := c.(*fluffycore_contracts_jwtminter.SigningKey)
			return signingKey
		}).ToSlice(&signingKeys)
		// return the last one.
		s.signingKey = signingKeys[len(signingKeys)-1]

		decrtypedPrivateKey, err := DecryptPEM([]byte(s.signingKey.PrivateKey), []byte(s.signingKey.Password))
		if err != nil {
			panic(err)
		}
		s.signingKey.DecryptedPrivateKey = string(decrtypedPrivateKey)
		/*
			// strip off the encryption and store the open key for downstream ease of use
			var method jwt.SigningMethod
			signingKey := s.signingKey
			switch signingKey.PrivateJwk.Alg {
			case "RS256":
				method = jwt.SigningMethodRS256
			case "RS384":
				method = jwt.SigningMethodRS384
			case "RS512":
				method = jwt.SigningMethodRS512
			case "ES256":
				method = jwt.SigningMethodES256
			case "ES384":
				method = jwt.SigningMethodES384
			case "ES512":
				method = jwt.SigningMethodES512
			case "EdDSA":
				method = jwt.SigningMethodEdDSA
			default:
				panic("unsupported signing method")
			}
			signedKey := []byte(signingKey.PrivateKey)

			var getKey = func() (interface{}, error) {
				var key interface{}

				if strings.HasPrefix(signingKey.PrivateJwk.Alg, "Ed") {
					v, err := jwt.ParseEdPrivateKeyFromPEM(signedKey)
					if err != nil {
						return "", err
					}
					key = v
					return key, nil
				}

				if strings.HasPrefix(signingKey.PrivateJwk.Alg, "ES") {
					v, err := jwt.ParseECPrivateKeyFromPEM(signedKey)
					if err != nil {
						return "", err
					}
					key = v
					return key, nil
				}

				v, err := jwt.ParseRSAPrivateKeyFromPEM(signedKey)
				if err != nil {
					return "", err
				}
				key = v
				return key, nil
			}

			s.signingKey.PrivateKey = signingKey.PrivateKey
		*/
		var jwks []*fluffycore_contracts_jwtminter.PublicJwk
		linq.From(s.keyMaterial.SigningKeys).Where(func(c interface{}) bool {
			signingKey := c.(*fluffycore_contracts_jwtminter.SigningKey)
			if now.After(signingKey.NotBefore) && now.Before(signingKey.NotAfter) {
				return true
			}
			return false
		}).Select(func(c interface{}) interface{} {
			signingKey := c.(*fluffycore_contracts_jwtminter.SigningKey)
			return &signingKey.PublicJwk
		}).ToSlice(&jwks)
		s.jwks = jwks
	}
}

func (s *service) GetSigningKey() (*fluffycore_contracts_jwtminter.SigningKey, error) {
	s._reloadKeys()
	//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
	s.lock.RLock()
	defer s.lock.RUnlock()
	//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--

	return s.signingKey, nil
}
func (s *service) GetSigningKeys() ([]*fluffycore_contracts_jwtminter.SigningKey, error) {
	s._reloadKeys()
	//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
	s.lock.RLock()
	defer s.lock.RUnlock()
	//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--

	return s.keyMaterial.SigningKeys, nil
}

func (s *service) CreateKeySet() (jwk.Set, error) {
	keys, err := s.GetSigningKeys()
	if err != nil {
		return nil, err
	}
	set := jwk.NewSet()
	for _, key := range keys {
		keyB, _ := json.Marshal(key.PrivateJwk)
		privkey, err := jwk.ParseKey(keyB)
		if err != nil {
			return nil, err
		}
		pubkey, err := jwk.PublicKeyOf(privkey)
		if err != nil {
			return nil, err
		}
		set.AddKey(pubkey)
	}
	return set, nil
}

func (s *service) GetPublicWebKeys() ([]*fluffycore_contracts_jwtminter.PublicJwk, error) {
	s._reloadKeys()
	//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
	s.lock.RLock()
	defer s.lock.RUnlock()
	//--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
	return s.jwks, nil
}

func DecryptPEM(encryptedPEM []byte, password []byte) ([]byte, error) {
	// Parse PEM block
	block, _ := pem.Decode(encryptedPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Check if encrypted
	if block.Headers["Proc-Type"] != "4,ENCRYPTED" {
		return encryptedPEM, nil // Return original if not encrypted
	}

	// Parse DEK-Info header
	dekInfo := strings.Split(block.Headers["DEK-Info"], ",")
	if len(dekInfo) != 2 {
		return nil, fmt.Errorf("malformed DEK-Info header")
	}

	// Get cipher type and IV
	cipherType := dekInfo[0]
	if cipherType != "AES-256-CBC" {
		return nil, fmt.Errorf("unsupported cipher: %s", cipherType)
	}

	// Decode IV from hex
	iv, err := hex.DecodeString(dekInfo[1])
	if err != nil {
		return nil, fmt.Errorf("invalid IV: %v", err)
	}

	// Generate key from password and IV
	key := generateKeyFromPassword(password, iv[:8], 32) // AES-256 needs 32 bytes

	// Create cipher
	block_cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Decrypt
	mode := cipher.NewCBCDecrypter(block_cipher, iv)
	decrypted := make([]byte, len(block.Bytes))
	mode.CryptBlocks(decrypted, block.Bytes)

	// Remove PKCS7 padding
	paddingLen := int(decrypted[len(decrypted)-1])
	decrypted = decrypted[:len(decrypted)-paddingLen]

	// Encode decrypted key in PEM format
	decryptedBlock := &pem.Block{
		Type:  block.Type,
		Bytes: decrypted,
	}

	// Encode to PEM
	out := new(bytes.Buffer)
	err = pem.Encode(out, decryptedBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to encode decrypted key: %v", err)
	}

	return out.Bytes(), nil
}

// Helper function to generate key from password using OpenSSL's EVP_BytesToKey
func generateKeyFromPassword(password, salt []byte, keyLen int) []byte {
	var result []byte
	hash := md5.New()

	// First iteration
	hash.Write(password)
	hash.Write(salt)
	result = hash.Sum(nil)

	// Subsequent iterations
	for len(result) < keyLen {
		hash.Reset()
		hash.Write(result)
		hash.Write(password)
		hash.Write(salt)
		result = append(result, hash.Sum(nil)...)
	}

	return result[:keyLen]
}
