package keymaterial

import (
	"encoding/json"
	"sync"
	"time"

	linq "github.com/ahmetb/go-linq/v3"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_jwtminter "github.com/fluffy-bunny/fluffycore/contracts/jwtminter"
	ecdsa "github.com/fluffy-bunny/fluffycore/utils/ecdsa"
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

func init() {
	var _ fluffycore_contracts_jwtminter.IKeyMaterial = stemService
}

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

		// strip off the encryption and store the open key for downstream ease of use
		privateKey, publicKey, err := ecdsa.DecodePrivatePem(s.signingKey.Password, s.signingKey.PrivateKey)
		if err != nil {
			panic(err)
		}
		encPriv, _, err := ecdsa.Encode("", privateKey, publicKey)
		s.signingKey.PrivateKey = encPriv

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
