package jwtminter

import (
	"strings"
	"time"
)

const signingKeysTemplate = `{
    "signing_keys": [
        {
            "private_key": "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIFA+8y3M5qxkjuI7HOUAPVgrsjUnu5kwRjsZlbCmyabCoAoGCCqGSM49\nAwEHoUQDQgAEYMrUm/S5+d+euQHrrzQMWJSFafSYcgUE0RYjfI7sErK75hSdE0aj\nPNMXaaDG395zD18VxjsmqPTWom17ncVnnw==\n-----END EC PRIVATE KEY-----\n",
            "public_key": "-----BEGIN EC  PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEYMrUm/S5+d+euQHrrzQMWJSFafSY\ncgUE0RYjfI7sErK75hSdE0ajPNMXaaDG395zD18VxjsmqPTWom17ncVnnw==\n-----END EC  PUBLIC KEY-----\n",
            "not_before": "{not_before}",
            "not_after": "{not_after}",
            "password": "",
            "kid": "0b2cd2e54c924ce89f010f242862367d",
            "public_jwk": {
                "alg": "ES256",
                "crv": "P-256",
                "kid": "0b2cd2e54c924ce89f010f242862367d",
                "kty": "EC",
                "use": "sig",
                "x": "YMrUm_S5-d-euQHrrzQMWJSFafSYcgUE0RYjfI7sErI",
                "y": "u-YUnRNGozzTF2mgxt_ecw9fFcY7Jqj01qJte53FZ58"
            },
            "private_jwk": {
                "alg": "ES256",
                "crv": "P-256",
                "d": "UD7zLczmrGSO4jsc5QA9WCuyNSe7mTBGOxmVsKbJpsI",
                "kid": "0b2cd2e54c924ce89f010f242862367d",
                "kty": "EC",
                "use": "sig",
                "x": "YMrUm_S5-d-euQHrrzQMWJSFafSYcgUE0RYjfI7sErI",
                "y": "u-YUnRNGozzTF2mgxt_ecw9fFcY7Jqj01qJte53FZ58"
            }
        }
    ]
}`

var signingKeys = ""

func GetSigningKeysJSON() string {
	now := time.Now()
	notBefore := now.Add(-1 * time.Hour)
	notAfter := now.Add(24 * time.Hour)

	nbf := notBefore.Format("2006-01-02T15:04:05Z")
	naf := notAfter.Format("2006-01-02T15:04:05Z")
	signingKeys = strings.Replace(signingKeysTemplate, "{not_before}", nbf, -1)
	signingKeys = strings.Replace(signingKeys, "{not_after}", naf, -1)
	return signingKeys

}
