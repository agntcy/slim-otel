// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"fmt"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
)

// AuthConfig represents the authentication configuration for SLIM applications
type AuthConfig struct {
	SharedSecret *string          `mapstructure:"shared-secret"`
	StaticJWT    *StaticJWTConfig `mapstructure:"static-jwt"`
	JWT          *JWTConfig       `mapstructure:"jwt"`
	Spire        *SpireConfig     `mapstructure:"spire"`
}

// StaticJWTConfig represents static JWT authentication configuration
type StaticJWTConfig struct {
	File     string `mapstructure:"file"`
	Duration string `mapstructure:"duration"`
}

// JWTConfig represents dynamic JWT authentication configuration
type JWTConfig struct {
	Claims   JWTClaims  `mapstructure:"claims"`
	Duration string     `mapstructure:"duration"`
	Key      JWTKeyEnum `mapstructure:"key"`
}

// JWTClaims represents JWT claims configuration
type JWTClaims struct {
	Audience []string `mapstructure:"audience"`
	Issuer   string   `mapstructure:"issuer"`
	Subject  string   `mapstructure:"subject"`
}

// JWTKeyEnum represents the JWT key configuration with type discrimination
type JWTKeyEnum struct {
	Encoding    *JWTKeyConfig `mapstructure:"encoding"`
	Decoding    *JWTKeyConfig `mapstructure:"decoding"`
	Autoresolve bool          `mapstructure:"autoresolve"`
}

// JWTKeyConfig represents the key configuration for JWT signing/verification
type JWTKeyConfig struct {
	Algorithm string       `mapstructure:"algorithm"`
	Format    string       `mapstructure:"format"`
	Key       JWTKeySource `mapstructure:"key"`
}

// JWTKeySource represents where the key data comes from
type JWTKeySource struct {
	File *string `mapstructure:"file"`
	Data *string `mapstructure:"data"`
}

// SpireConfig represents SPIRE authentication configuration
type SpireConfig struct {
	SocketPath     string   `mapstructure:"socket-path"`
	TargetSpiffeID string   `mapstructure:"target-spiffe-id"`
	JWTAudiences   []string `mapstructure:"jwt-audiences"`
	TrustDomains   []string `mapstructure:"trust-domains"`
}

// ToIdentityProviderConfig converts AuthConfig to SLIM's IdentityProviderConfig
func (a *AuthConfig) ToIdentityProviderConfig(appName string) (slim.IdentityProviderConfig, error) {
	// Count how many auth methods are configured
	configured := 0
	if a.SharedSecret != nil {
		configured++
	}
	if a.StaticJWT != nil {
		configured++
	}
	if a.JWT != nil {
		configured++
	}
	if a.Spire != nil {
		configured++
	}

	if configured == 0 {
		return nil, fmt.Errorf("no authentication method configured")
	}
	if configured > 1 {
		return nil, fmt.Errorf("only one authentication method can be configured at a time")
	}

	// Convert based on which method is configured
	if a.SharedSecret != nil {
		return slim.NewIdentityProviderConfigSharedSecret(appName, *a.SharedSecret), nil
	}

	if a.StaticJWT != nil {
		duration, err := parseDuration(a.StaticJWT.Duration, 3600*time.Second)
		if err != nil {
			return nil, fmt.Errorf("invalid static JWT duration: %w", err)
		}

		staticJWTAuth := slim.NewStaticJwtAuth(a.StaticJWT.File, duration)
		return slim.NewIdentityProviderConfigStaticJwt(staticJWTAuth), nil
	}

	if a.JWT != nil {
		return a.jwtToProviderConfig()
	}

	if a.Spire != nil {
		return a.spireToProviderConfig()
	}

	return nil, fmt.Errorf("no valid authentication method found")
}

// jwtToProviderConfig converts JWT config to IdentityProviderConfig
func (a *AuthConfig) jwtToProviderConfig() (slim.IdentityProviderConfig, error) {
	duration, err := parseDuration(a.JWT.Duration, 3600*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT duration: %w", err)
	}

	// Determine key type
	var keyType slim.JwtKeyType
	if a.JWT.Key.Encoding != nil {
		keyConfig, err := a.JWT.Key.Encoding.toSlimJWTKeyConfig()
		if err != nil {
			return nil, fmt.Errorf("invalid encoding key config: %w", err)
		}
		keyType = slim.NewJwtKeyTypeEncoding(keyConfig)
	} else if a.JWT.Key.Decoding != nil {
		keyConfig, err := a.JWT.Key.Decoding.toSlimJWTKeyConfig()
		if err != nil {
			return nil, fmt.Errorf("invalid decoding key config: %w", err)
		}
		keyType = slim.NewJwtKeyTypeDecoding(keyConfig)
	} else if a.JWT.Key.Autoresolve {
		keyType = slim.NewJwtKeyTypeAutoresolve()
	} else {
		return nil, fmt.Errorf("JWT key must specify encoding, decoding, or autoresolve")
	}

	// Convert audiences to pointers if present
	var audiences *[]string
	if len(a.JWT.Claims.Audience) > 0 {
		audiences = &a.JWT.Claims.Audience
	}

	// Convert issuer and subject to pointers if present
	var issuer, subject *string
	if a.JWT.Claims.Issuer != "" {
		issuer = &a.JWT.Claims.Issuer
	}
	if a.JWT.Claims.Subject != "" {
		subject = &a.JWT.Claims.Subject
	}

	clientJWTAuth := slim.NewClientJwtAuth(keyType, audiences, issuer, subject, duration)
	return slim.NewIdentityProviderConfigJwt(clientJWTAuth), nil
}

// spireToProviderConfig converts SPIRE config to IdentityProviderConfig
func (a *AuthConfig) spireToProviderConfig() (slim.IdentityProviderConfig, error) {
	var socketPath, targetSpiffeID *string
	var jwtAudiences, trustDomains []string

	if a.Spire.SocketPath != "" {
		socketPath = &a.Spire.SocketPath
	}
	if a.Spire.TargetSpiffeID != "" {
		targetSpiffeID = &a.Spire.TargetSpiffeID
	}
	if len(a.Spire.JWTAudiences) > 0 {
		jwtAudiences = a.Spire.JWTAudiences
	}
	if len(a.Spire.TrustDomains) > 0 {
		trustDomains = a.Spire.TrustDomains
	}

	spireConfig := slim.NewSpireConfig(socketPath, targetSpiffeID, jwtAudiences, trustDomains)
	return slim.NewIdentityProviderConfigSpire(spireConfig), nil
}

// toSlimJWTKeyConfig converts JWTKeyConfig to SLIM's JwtKeyConfig
func (k *JWTKeyConfig) toSlimJWTKeyConfig() (slim.JwtKeyConfig, error) {
	// Parse algorithm
	algorithm, err := parseJWTAlgorithm(k.Algorithm)
	if err != nil {
		return slim.JwtKeyConfig{}, err
	}

	// Parse format
	format, err := parseJWTKeyFormat(k.Format)
	if err != nil {
		return slim.JwtKeyConfig{}, err
	}

	// Parse key source
	var keyData slim.JwtKeyData
	if k.Key.File != nil {
		keyData = slim.NewJwtKeyDataFile(*k.Key.File)
	} else if k.Key.Data != nil {
		keyData = slim.NewJwtKeyDataData(*k.Key.Data)
	} else {
		return slim.JwtKeyConfig{}, fmt.Errorf("JWT key must specify either 'file' or 'data'")
	}

	return slim.NewJwtKeyConfig(algorithm, format, keyData), nil
}

// parseJWTAlgorithm converts string to JwtAlgorithm enum
func parseJWTAlgorithm(alg string) (slim.JwtAlgorithm, error) {
	switch alg {
	case "HS256":
		return slim.JwtAlgorithmHS256, nil
	case "HS384":
		return slim.JwtAlgorithmHS384, nil
	case "HS512":
		return slim.JwtAlgorithmHS512, nil
	case "ES256":
		return slim.JwtAlgorithmES256, nil
	case "ES384":
		return slim.JwtAlgorithmES384, nil
	case "RS256":
		return slim.JwtAlgorithmRS256, nil
	case "RS384":
		return slim.JwtAlgorithmRS384, nil
	case "RS512":
		return slim.JwtAlgorithmRS512, nil
	case "PS256":
		return slim.JwtAlgorithmPS256, nil
	case "PS384":
		return slim.JwtAlgorithmPS384, nil
	case "PS512":
		return slim.JwtAlgorithmPS512, nil
	case "EdDSA":
		return slim.JwtAlgorithmEdDSA, nil
	default:
		return 0, fmt.Errorf("unsupported JWT algorithm: %s", alg)
	}
}

// parseJWTKeyFormat converts string to JwtKeyFormat enum
func parseJWTKeyFormat(format string) (slim.JwtKeyFormat, error) {
	switch format {
	case "pem":
		return slim.JwtKeyFormatPem, nil
	case "jwk":
		return slim.JwtKeyFormatJwk, nil
	case "jwks":
		return slim.JwtKeyFormatJwks, nil
	default:
		return 0, fmt.Errorf("unsupported JWT key format: %s", format)
	}
}

// parseDuration parses a duration string with a default value
func parseDuration(durationStr string, defaultDuration time.Duration) (time.Duration, error) {
	if durationStr == "" {
		return defaultDuration, nil
	}
	return time.ParseDuration(durationStr)
}
