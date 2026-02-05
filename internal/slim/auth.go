// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"fmt"
	"strings"
	"time"

	slim "github.com/agntcy/slim/bindings/generated/slim_bindings"
)

// AuthConfig represents the authentication configuration for SLIM applications
type AuthConfig struct {
	SharedSecret *string      `mapstructure:"shared-secret"`
	JWT          *JWTConfig   `mapstructure:"jwt"`
	Spire        *SpireConfig `mapstructure:"spire"`
}

// JWTConfig represents dynamic JWT authentication configuration
type JWTConfig struct {
	Claims   JWTClaims `mapstructure:"claims"`
	Duration string    `mapstructure:"duration"`
	Key      JWTKey    `mapstructure:"key"`
}

// JWTClaims represents JWT claims configuration
type JWTClaims struct {
	Audience []string `mapstructure:"audience"`
	Issuer   string   `mapstructure:"issuer"`
	Subject  string   `mapstructure:"subject"`
}

// JWTKey represents the JWT key configuration with type (encoding/decoding/autoresolve)
type JWTKey struct {
	Encoding *JWTKeyConfig `mapstructure:"encoding"`
	Decoding *JWTKeyConfig `mapstructure:"decoding"`
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
	if err := a.ValidateAuthConfig(); err != nil {
		return nil, err
	}

	// Convert based on which method is configured
	if a.SharedSecret != nil {
		return slim.IdentityProviderConfigSharedSecret{
			Data: *a.SharedSecret,
			Id:   appName,
		}, nil
	}

	if a.JWT != nil {
		return a.jwtToProviderConfig()
	}

	if a.Spire != nil {
		return a.spireToProviderConfig()
	}

	return nil, fmt.Errorf("no valid authentication method found")
}

// ToIdentityVerifierConfig converts AuthConfig to SLIM's IdentityVerifierConfig
func (a *AuthConfig) ToIdentityVerifierConfig(appName string) (slim.IdentityVerifierConfig, error) {
	if err := a.ValidateAuthConfig(); err != nil {
		return nil, err
	}

	// Convert based on which method is configured
	if a.SharedSecret != nil {
		return slim.IdentityVerifierConfigSharedSecret{
			Data: *a.SharedSecret,
			Id:   appName,
		}, nil
	}

	if a.JWT != nil {
		return a.jwtToVerifierConfig()
	}

	if a.Spire != nil {
		return a.spireToVerifierConfig()
	}

	return nil, fmt.Errorf("no valid authentication method found")
}

// jwtToProviderConfig converts JWT config to IdentityProviderConfig
func (a *AuthConfig) jwtToProviderConfig() (slim.IdentityProviderConfig, error) {
	duration, err := parseDuration(a.JWT.Duration, 3600*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT duration: %w", err)
	}

	// Get encoding key configuration (for signing)
	if a.JWT.Key.Encoding == nil {
		return nil, fmt.Errorf("JWT key encoding configuration is required for IdentityProvider")
	}

	keyConfig, err := a.JWT.Key.Encoding.toJWTKeyConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid encoding key config: %w", err)
	}
	keyType := slim.JwtKeyTypeEncoding{Key: keyConfig}

	audiences, issuer, subject := a.convertJWTClaims()

	return slim.IdentityProviderConfigJwt{
		Config: slim.ClientJwtAuth{
			Key:      keyType,
			Audience: audiences,
			Issuer:   issuer,
			Subject:  subject,
			Duration: duration,
		},
	}, nil
}

// jwtToVerifierConfig converts JWT config to IdentityVerifierConfig
func (a *AuthConfig) jwtToVerifierConfig() (slim.IdentityVerifierConfig, error) {
	duration, err := parseDuration(a.JWT.Duration, 3600*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT duration: %w", err)
	}

	var keyType slim.JwtKeyType
	// Check if decoding key is provided
	if a.JWT.Key.Decoding != nil {
		keyConfig, err := a.JWT.Key.Decoding.toJWTKeyConfig()
		if err != nil {
			return nil, fmt.Errorf("invalid decoding key config: %w", err)
		}
		keyType = slim.JwtKeyTypeDecoding{Key: keyConfig}
	} else {
		// Default to autoresolve if no decoding key is specified
		keyType = slim.JwtKeyTypeAutoresolve{}
	}

	audiences, issuer, subject := a.convertJWTClaims()

	return slim.IdentityVerifierConfigJwt{
		Config: slim.JwtAuth{
			Key:      keyType,
			Audience: audiences,
			Issuer:   issuer,
			Subject:  subject,
			Duration: duration,
		},
	}, nil
}

// spireToProviderConfig converts SPIRE config to IdentityProviderConfig
func (a *AuthConfig) spireToProviderConfig() (slim.IdentityProviderConfig, error) {
	return slim.IdentityProviderConfigSpire{
		Config: a.prepareSpireConfig(),
	}, nil
}

// spireToVerifierConfig converts SPIRE config to IdentityVerifierConfig
func (a *AuthConfig) spireToVerifierConfig() (slim.IdentityVerifierConfig, error) {
	return slim.IdentityVerifierConfigSpire{
		Config: a.prepareSpireConfig(),
	}, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// ValidateAuthConfig validates that exactly one authentication method is configured
func (a *AuthConfig) ValidateAuthConfig() error {
	configured := 0
	if a.SharedSecret != nil {
		configured++
	}
	if a.JWT != nil {
		configured++
	}
	if a.Spire != nil {
		configured++
	}

	if configured == 0 {
		return fmt.Errorf("no authentication method configured")
	}
	if configured > 1 {
		return fmt.Errorf("only one authentication method can be configured at a time")
	}
	return nil
}

// convertJWTClaims converts JWT claims to pointers for the SLIM API
func (a *AuthConfig) convertJWTClaims() (*[]string, *string, *string) {
	var audiences *[]string
	if len(a.JWT.Claims.Audience) > 0 {
		audiences = &a.JWT.Claims.Audience
	}

	var issuer, subject *string
	if a.JWT.Claims.Issuer != "" {
		issuer = &a.JWT.Claims.Issuer
	}
	if a.JWT.Claims.Subject != "" {
		subject = &a.JWT.Claims.Subject
	}

	return audiences, issuer, subject
}

// prepareSpireConfig prepares SPIRE configuration
func (a *AuthConfig) prepareSpireConfig() slim.SpireConfig {
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

	return slim.SpireConfig{
		SocketPath:     socketPath,
		TargetSpiffeId: targetSpiffeID,
		JwtAudiences:   jwtAudiences,
		TrustDomains:   trustDomains,
	}
}

// toJWTKeyConfig converts JWTKeyConfig to SLIM's JwtKeyConfig
func (k *JWTKeyConfig) toJWTKeyConfig() (slim.JwtKeyConfig, error) {
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
		keyData = slim.JwtKeyDataFile{Path: *k.Key.File}
	} else if k.Key.Data != nil {
		keyData = slim.JwtKeyDataData{Value: *k.Key.Data}
	} else {
		return slim.JwtKeyConfig{}, fmt.Errorf("JWT key must specify either 'file' or 'data'")
	}

	return slim.JwtKeyConfig{
		Algorithm: algorithm,
		Format:    format,
		Key:       keyData,
	}, nil
}

// parseJWTAlgorithm converts string to slim.`JwtAlgorithm enum
func parseJWTAlgorithm(alg string) (slim.JwtAlgorithm, error) {
	switch alg {
	case "HS256":
		return slim.JwtAlgorithmHs256, nil
	case "HS384":
		return slim.JwtAlgorithmHs384, nil
	case "HS512":
		return slim.JwtAlgorithmHs512, nil
	case "ES256":
		return slim.JwtAlgorithmEs256, nil
	case "ES384":
		return slim.JwtAlgorithmEs384, nil
	case "RS256":
		return slim.JwtAlgorithmRs256, nil
	case "RS384":
		return slim.JwtAlgorithmRs384, nil
	case "RS512":
		return slim.JwtAlgorithmRs512, nil
	case "PS256":
		return slim.JwtAlgorithmPs256, nil
	case "PS384":
		return slim.JwtAlgorithmPs384, nil
	case "PS512":
		return slim.JwtAlgorithmPs512, nil
	case "EdDSA":
		return slim.JwtAlgorithmEdDsa, nil
	default:
		return 0, fmt.Errorf("unsupported JWT algorithm: %s", alg)
	}
}

// parseJWTKeyFormat converts string to slim.JwtKeyFormat enum
func parseJWTKeyFormat(format string) (slim.JwtKeyFormat, error) {
	// Convert to lowercase for case-insensitive comparison
	switch strings.ToLower(format) {
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
