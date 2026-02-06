// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"errors"
	"fmt"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
)

// ConnectionConfig defines the SLIM endpoint connection configuration
type ConnectionConfig struct {
	// Address of the SLIM endpoint to connect to
	Address string `mapstructure:"address"`

	// Origin header value (optional)
	Origin *string `mapstructure:"origin"`

	// Server name for TLS verification (optional)
	ServerName *string `mapstructure:"server_name"`

	// Compression algorithm: "gzip", "zlib", "deflate", "snappy", "zstd", "lz4", "none", "empty" (optional)
	Compression *string `mapstructure:"compression"`

	// Rate limit for requests (e.g., "100/s", "1000/m") (optional)
	RateLimit *string `mapstructure:"rate_limit"`

	// TLS configuration
	TLS *TLSConfig `mapstructure:"tls"`

	// Keepalive configuration
	Keepalive *KeepaliveConfig `mapstructure:"keepalive"`

	// Proxy configuration
	Proxy *ProxyConfig `mapstructure:"proxy"`

	// Connection timeout
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`

	// Request timeout
	RequestTimeout time.Duration `mapstructure:"request_timeout"`

	// Buffer size in bytes
	BufferSize *uint64 `mapstructure:"buffer_size"`

	// Additional headers to include in requests
	Headers map[string]string `mapstructure:"headers"`

	// Authentication configuration
	Auth *AuthConfig `mapstructure:"auth"`

	// Backoff configuration for retries
	Backoff *BackoffConfig `mapstructure:"backoff"`

	// Metadata for the connection (optional)
	Metadata *string `mapstructure:"metadata"`
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	// Set to true for insecure connections (no TLS)
	Insecure bool `mapstructure:"insecure"`

	// CA source configuration for verifying server certificates (client side)
	CASource *TLSCAConfig `mapstructure:"ca_source"`

	// Client certificate and key for mTLS
	Source *TLSCertKeySource `mapstructure:"source"`

	// TLS version constraint: "tls1.2" or "tls1.3" (default: "tls1.3")
	TLSVersion string `mapstructure:"tls_version"`

	// Include system CA certificates pool (default: true)
	IncludeSystemCACertsPool bool `mapstructure:"include_system_ca_certs_pool"`

	// Skip server name verification (INSECURE, default: false)
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// TLSCAConfig defines CA certificate configuration
// Only one of Path or Data should be set
type TLSCAConfig struct {
	// Path to the CA certificate file
	Path *string `mapstructure:"path"`

	// PEM encoded CA certificate data
	Data *string `mapstructure:"data"`
}

// TLSCertKeySource defines client certificate and key configuration for mTLS
// Supports both file paths and inline PEM data
type TLSCertKeySource struct {
	// Path to the certificate file (for file-based source)
	CertFile *string `mapstructure:"cert_file"`

	// Path to the key file (for file-based source)
	KeyFile *string `mapstructure:"key_file"`

	// PEM encoded certificate data (for PEM-based source)
	CertData *string `mapstructure:"cert_data"`

	// PEM encoded key data (for PEM-based source)
	KeyData *string `mapstructure:"key_data"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	// Type of authentication: "basic", "jwt", "static_jwt", or "none"
	Type string `mapstructure:"type"`

	// Basic authentication configuration
	Basic *BasicAuthConfig `mapstructure:"basic"`

	// Static JWT authentication configuration
	StaticJwt *StaticJwtAuthConfig `mapstructure:"static_jwt"`

	// JWT authentication configuration
	Jwt *JwtAuthConfig `mapstructure:"jwt"`
}

// BasicAuthConfig defines basic authentication configuration
type BasicAuthConfig struct {
	// Username for basic authentication
	Username string `mapstructure:"username"`

	// Password for basic authentication
	Password string `mapstructure:"password"`
}

// StaticJwtAuthConfig defines static JWT authentication configuration
type StaticJwtAuthConfig struct {
	// Path to file containing the JWT token
	TokenFile string `mapstructure:"token_file"`

	// Duration for caching the token before re-reading from file
	Duration time.Duration `mapstructure:"duration"`
}

// JwtAuthConfig defines JWT authentication configuration
type JwtAuthConfig struct {
	// Token validity duration
	Duration time.Duration `mapstructure:"duration"`

	// JWT audience claims to include
	Audience []string `mapstructure:"audience"`

	// JWT issuer to include
	Issuer string `mapstructure:"issuer"`

	// JWT subject to include
	Subject string `mapstructure:"subject"`

	// JWT key configuration
	Key *JWTKeyConfig `mapstructure:"key"`
}

// JWTKeyConfig defines JWT key configuration
type JWTKeyConfig struct {
	// Algorithm used for encoding: ES256, RS256, HS256, etc.
	Algorithm string `mapstructure:"algorithm"`

	// Format of the key: "pem"
	Format string `mapstructure:"format"`

	// Key source (file or inline data)
	Key *JWTKeySource `mapstructure:"key"`
}

// JWTKeySource defines the source of the JWT key
type JWTKeySource struct {
	// Path to a file containing the key
	File string `mapstructure:"file"`

	// Inline key data (PEM encoded private key or secret)
	Data string `mapstructure:"data"`
}

// KeepaliveConfig defines keepalive configuration
type KeepaliveConfig struct {
	// TCP keepalive duration
	TcpKeepalive time.Duration `mapstructure:"tcp_keepalive"`

	// HTTP/2 keepalive duration
	Http2Keepalive time.Duration `mapstructure:"http2_keepalive"`

	// Keepalive timeout
	Timeout time.Duration `mapstructure:"timeout"`

	// Whether to permit keepalive without an active stream
	KeepAliveWhileIdle bool `mapstructure:"keep_alive_while_idle"`
}

// ProxyConfig defines proxy configuration
type ProxyConfig struct {
	// URL of the proxy server (optional)
	Url *string `mapstructure:"url"`

	// TLS configuration for the proxy connection
	Tls *TLSConfig `mapstructure:"tls"`

	// Username for proxy authentication (optional)
	Username *string `mapstructure:"username"`

	// Password for proxy authentication (optional)
	Password *string `mapstructure:"password"`

	// Additional headers to send to the proxy
	Headers map[string]string `mapstructure:"headers"`
}

// BackoffConfig defines backoff configuration for retries
type BackoffConfig struct {
	// Type of backoff: "exponential" or "fixed_interval"
	Type string `mapstructure:"type"`

	// Exponential backoff configuration
	Exponential *ExponentialBackoffConfig `mapstructure:"exponential"`

	// Fixed interval backoff configuration
	FixedInterval *FixedIntervalBackoffConfig `mapstructure:"fixed_interval"`
}

// ExponentialBackoffConfig defines exponential backoff configuration
type ExponentialBackoffConfig struct {
	// Base duration for exponential backoff
	Base time.Duration `mapstructure:"base"`

	// Multiplication factor for exponential backoff
	Factor uint64 `mapstructure:"factor"`

	// Maximum delay between retries
	MaxDelay time.Duration `mapstructure:"max_delay"`

	// Maximum number of retry attempts
	MaxAttempts uint64 `mapstructure:"max_attempts"`

	// Enable jitter for exponential backoff
	Jitter bool `mapstructure:"jitter"`
}

// FixedIntervalBackoffConfig defines fixed interval backoff configuration
type FixedIntervalBackoffConfig struct {
	// Interval for fixed interval backoff
	Interval time.Duration `mapstructure:"interval"`

	// Maximum number of retry attempts
	MaxAttempts uint64 `mapstructure:"max_attempts"`
}

// Validate checks if the connection configuration is valid
func (cfg *ConnectionConfig) Validate() error {
	if cfg.Address == "" {
		return errors.New("connection address is required")
	}

	// Validate TLS configuration
	if cfg.TLS != nil {
		if err := validateTLSConfig(cfg.TLS); err != nil {
			return fmt.Errorf("invalid TLS config: %w", err)
		}
	}

	// Validate authentication configuration
	if cfg.Auth != nil {
		if err := validateAuthConfig(cfg.Auth); err != nil {
			return fmt.Errorf("invalid auth config: %w", err)
		}
	}

	// Validate compression type
	if cfg.Compression != nil {
		if err := validateCompression(*cfg.Compression); err != nil {
			return err
		}
	}

	// Validate backoff configuration
	if cfg.Backoff != nil {
		if err := validateBackoffConfig(cfg.Backoff); err != nil {
			return fmt.Errorf("invalid backoff config: %w", err)
		}
	}

	// Validate proxy configuration
	if cfg.Proxy != nil {
		if err := validateProxyConfig(cfg.Proxy); err != nil {
			return fmt.Errorf("invalid proxy config: %w", err)
		}
	}

	return nil
}

// validateTLSConfig validates TLS configuration
func validateTLSConfig(cfg *TLSConfig) error {
	// Validate TLS version
	if cfg.TLSVersion != "" {
		if cfg.TLSVersion != "tls1.2" && cfg.TLSVersion != "tls1.3" {
			return fmt.Errorf("invalid TLS version: %s (must be 'tls1.2' or 'tls1.3')", cfg.TLSVersion)
		}
	}

	// Validate CA source configuration
	if !cfg.Insecure && cfg.CASource != nil {
		if err := validateCAConfig(cfg.CASource); err != nil {
			return fmt.Errorf("invalid CA source: %w", err)
		}
	}

	// Validate mTLS configuration if provided
	if cfg.Source != nil {
		if err := validateTLSSource(cfg.Source); err != nil {
			return fmt.Errorf("invalid TLS source: %w", err)
		}
	}

	return nil
}

// validateCAConfig validates CA certificate configuration
func validateCAConfig(ca *TLSCAConfig) error {
	if ca.Path != nil && ca.Data != nil {
		return errors.New("CA path and data cannot both be specified")
	}

	if ca.Path != nil && *ca.Path == "" {
		return errors.New("CA path cannot be empty")
	}

	if ca.Data != nil && *ca.Data == "" {
		return errors.New("CA data cannot be empty")
	}

	return nil
}

// validateTLSSource validates TLS certificate and key source
func validateTLSSource(cfg *TLSCertKeySource) error {
	hasFile := cfg.CertFile != nil || cfg.KeyFile != nil
	hasPEM := cfg.CertData != nil || cfg.KeyData != nil

	if hasFile && hasPEM {
		return errors.New("cannot specify both file and PEM sources for TLS certificate")
	}

	if hasFile {
		if cfg.CertFile == nil || *cfg.CertFile == "" {
			return errors.New("cert_file is required for file-based TLS source")
		}
		if cfg.KeyFile == nil || *cfg.KeyFile == "" {
			return errors.New("key_file is required for file-based TLS source")
		}
	} else if hasPEM {
		if cfg.CertData == nil || *cfg.CertData == "" {
			return errors.New("cert_data is required for PEM-based TLS source")
		}
		if cfg.KeyData == nil || *cfg.KeyData == "" {
			return errors.New("key_data is required for PEM-based TLS source")
		}
	} else {
		return errors.New("either file or PEM source must be specified for TLS certificate")
	}

	return nil
}

// validateAuthConfig validates authentication configuration
func validateAuthConfig(cfg *AuthConfig) error {
	if cfg.Type == "" {
		return errors.New("auth type is required")
	}

	switch cfg.Type {
	case "basic":
		if cfg.Basic == nil {
			return errors.New("basic auth configuration is required")
		}
		if cfg.Basic.Username == "" {
			return errors.New("username is required for basic authentication")
		}
		if cfg.Basic.Password == "" {
			return errors.New("password is required for basic authentication")
		}
	case "jwt":
		if cfg.Jwt == nil {
			return errors.New("JWT configuration is required")
		}
		if len(cfg.Jwt.Audience) == 0 {
			return errors.New("at least one audience is required for JWT authentication")
		}
		if cfg.Jwt.Key == nil {
			return errors.New("JWT key configuration is required for JWT authentication")
		}
		if cfg.Jwt.Key.Algorithm == "" {
			return errors.New("JWT key algorithm is required")
		}
		if cfg.Jwt.Key.Key == nil {
			return errors.New("JWT key source is required")
		}
		if cfg.Jwt.Key.Key.File == "" && cfg.Jwt.Key.Key.Data == "" {
			return errors.New("JWT key file or data is required")
		}
		if cfg.Jwt.Key.Key.File != "" && cfg.Jwt.Key.Key.Data != "" {
			return errors.New("JWT key file and data cannot both be specified")
		}
	case "static_jwt":
		if cfg.StaticJwt == nil {
			return errors.New("static JWT configuration is required")
		}
		if cfg.StaticJwt.TokenFile == "" {
			return errors.New("token file is required for static_jwt authentication")
		}
	case "none":
		// No validation needed
	default:
		return fmt.Errorf("unsupported auth type: %s", cfg.Type)
	}

	return nil
}

// validateCompression validates compression type
func validateCompression(compression string) error {
	validCompressions := []string{"gzip", "zlib", "deflate", "snappy", "zstd", "lz4", "none", "empty"}
	for _, v := range validCompressions {
		if compression == v {
			return nil
		}
	}
	return fmt.Errorf("invalid compression type: %s (must be one of: %v)", compression, validCompressions)
}

// validateBackoffConfig validates backoff configuration
func validateBackoffConfig(cfg *BackoffConfig) error {
	if cfg.Type == "" {
		return errors.New("backoff type is required")
	}

	switch cfg.Type {
	case "exponential":
		if cfg.Exponential == nil {
			return errors.New("exponential backoff configuration is required")
		}
		if cfg.Exponential.Base == 0 {
			return errors.New("base duration is required for exponential backoff")
		}
	case "fixed_interval":
		if cfg.FixedInterval == nil {
			return errors.New("fixed interval backoff configuration is required")
		}
		if cfg.FixedInterval.Interval == 0 {
			return errors.New("interval is required for fixed interval backoff")
		}
	default:
		return fmt.Errorf("invalid backoff type: %s (must be 'exponential' or 'fixed_interval')", cfg.Type)
	}

	return nil
}

// validateProxyConfig validates proxy configuration
func validateProxyConfig(cfg *ProxyConfig) error {
	if cfg.Url == nil && (cfg.Username != nil || cfg.Password != nil) {
		return errors.New("proxy URL is required when username or password is specified")
	}

	// Recursively validate proxy TLS configuration
	if cfg.Tls != nil {
		if err := validateTLSConfig(cfg.Tls); err != nil {
			return fmt.Errorf("invalid proxy TLS config: %w", err)
		}
	}

	return nil
}

// SetDefaults sets default values for the connection configuration
func (cfg *ConnectionConfig) SetDefaults() {
	if cfg.TLS == nil {
		cfg.TLS = &TLSConfig{
			Insecure: true,
		}
	}

	// Set TLS defaults
	if cfg.TLS != nil {
		if cfg.TLS.TLSVersion == "" {
			cfg.TLS.TLSVersion = "tls1.3"
		}
		// Default to including system CA certs pool
		if !cfg.TLS.Insecure && cfg.TLS.CASource != nil {
			cfg.TLS.IncludeSystemCACertsPool = true
		}
	}

	if cfg.Auth != nil && cfg.Auth.Type == "jwt" && cfg.Auth.Jwt != nil {
		if cfg.Auth.Jwt.Duration == 0 {
			cfg.Auth.Jwt.Duration = 1 * time.Hour
		}
		if cfg.Auth.Jwt.Key != nil && cfg.Auth.Jwt.Key.Format == "" {
			cfg.Auth.Jwt.Key.Format = "pem"
		}
	}
}

// ToSlimClientConfig converts the ConnectionConfig to a slim.ClientConfig
func (cfg *ConnectionConfig) ToSlimClientConfig() (slim.ClientConfig, error) {
	clientCfg := slim.ClientConfig{
		Endpoint:       cfg.Address,
		Origin:         cfg.Origin,
		ServerName:     cfg.ServerName,
		RateLimit:      cfg.RateLimit,
		ConnectTimeout: cfg.ConnectTimeout,
		RequestTimeout: cfg.RequestTimeout,
		BufferSize:     cfg.BufferSize,
		Headers:        cfg.Headers,
		Metadata:       cfg.Metadata,
	}

	// Convert compression
	if cfg.Compression != nil {
		compressionType, err := parseCompressionType(*cfg.Compression)
		if err != nil {
			return clientCfg, fmt.Errorf("invalid compression type: %w", err)
		}
		clientCfg.Compression = &compressionType
	}

	// Convert TLS configuration
	if cfg.TLS != nil {
		tlsCfg, err := cfg.TLS.toSlimTLSConfig()
		if err != nil {
			return clientCfg, fmt.Errorf("failed to convert TLS config: %w", err)
		}
		clientCfg.Tls = tlsCfg
	}

	// Convert keepalive configuration
	if cfg.Keepalive != nil {
		clientCfg.Keepalive = cfg.Keepalive.toSlimKeepaliveConfig()
	}

	// Convert proxy configuration
	if cfg.Proxy != nil {
		proxyCfg, err := cfg.Proxy.toSlimProxyConfig()
		if err != nil {
			return clientCfg, fmt.Errorf("failed to convert proxy config: %w", err)
		}
		clientCfg.Proxy = proxyCfg
	}

	// Convert authentication configuration
	if cfg.Auth != nil {
		authCfg, err := cfg.Auth.toSlimAuthConfig()
		if err != nil {
			return clientCfg, fmt.Errorf("failed to convert auth config: %w", err)
		}
		clientCfg.Auth = authCfg
	}

	// Convert backoff configuration
	if cfg.Backoff != nil {
		backoffCfg, err := cfg.Backoff.toSlimBackoffConfig()
		if err != nil {
			return clientCfg, fmt.Errorf("failed to convert backoff config: %w", err)
		}
		clientCfg.Backoff = backoffCfg
	}

	return clientCfg, nil
}

// parseCompressionType converts string compression type to slim.CompressionType
func parseCompressionType(compression string) (slim.CompressionType, error) {
	switch compression {
	case "gzip":
		return slim.CompressionTypeGzip, nil
	case "zlib":
		return slim.CompressionTypeZlib, nil
	case "deflate":
		return slim.CompressionTypeDeflate, nil
	case "snappy":
		return slim.CompressionTypeSnappy, nil
	case "zstd":
		return slim.CompressionTypeZstd, nil
	case "lz4":
		return slim.CompressionTypeLz4, nil
	case "none":
		return slim.CompressionTypeNone, nil
	case "empty":
		return slim.CompressionTypeEmpty, nil
	default:
		return 0, fmt.Errorf("unknown compression type: %s", compression)
	}
}

// toSlimTLSConfig converts TLSConfig to slim.TlsClientConfig
func (cfg *TLSConfig) toSlimTLSConfig() (slim.TlsClientConfig, error) {
	tlsCfg := slim.TlsClientConfig{
		Insecure:                 cfg.Insecure,
		InsecureSkipVerify:       cfg.InsecureSkipVerify,
		IncludeSystemCaCertsPool: cfg.IncludeSystemCACertsPool,
		TlsVersion:               cfg.TLSVersion,
	}

	// Convert CA source
	if cfg.CASource != nil {
		caSource, err := cfg.CASource.toSlimCASource()
		if err != nil {
			return tlsCfg, fmt.Errorf("failed to convert CA source: %w", err)
		}
		tlsCfg.CaSource = caSource
	}

	// Convert TLS source (client cert)
	if cfg.Source != nil {
		tlsSource, err := cfg.Source.toSlimTLSSource()
		if err != nil {
			return tlsCfg, fmt.Errorf("failed to convert TLS source: %w", err)
		}
		tlsCfg.Source = tlsSource
	}

	return tlsCfg, nil
}

// toSlimCASource converts TLSCAConfig to slim.CaSource interface
func (cfg *TLSCAConfig) toSlimCASource() (slim.CaSource, error) {
	if cfg.Path != nil && cfg.Data != nil {
		return nil, errors.New("CA path and data cannot both be specified")
	}

	if cfg.Path != nil {
		if *cfg.Path == "" {
			return nil, errors.New("CA file path cannot be empty")
		}
		return slim.CaSourceFile{Path: *cfg.Path}, nil
	}

	if cfg.Data != nil {
		if *cfg.Data == "" {
			return nil, errors.New("CA PEM data cannot be empty")
		}
		return slim.CaSourcePem{Data: *cfg.Data}, nil
	}

	// Both nil means no CA source (CaSourceNone)
	return slim.CaSourceNone{}, nil
}

// toSlimTLSSource converts TLSCertKeySource to slim.TlsSource interface
func (cfg *TLSCertKeySource) toSlimTLSSource() (slim.TlsSource, error) {
	hasFile := cfg.CertFile != nil || cfg.KeyFile != nil
	hasPEM := cfg.CertData != nil || cfg.KeyData != nil

	if hasFile && hasPEM {
		return nil, errors.New("cannot specify both file and PEM sources for TLS certificate")
	}

	if hasFile {
		if cfg.CertFile == nil || *cfg.CertFile == "" {
			return nil, errors.New("cert_file is required for file-based TLS source")
		}
		if cfg.KeyFile == nil || *cfg.KeyFile == "" {
			return nil, errors.New("key_file is required for file-based TLS source")
		}
		return slim.TlsSourceFile{
			Cert: *cfg.CertFile,
			Key:  *cfg.KeyFile,
		}, nil
	}

	if hasPEM {
		if cfg.CertData == nil || *cfg.CertData == "" {
			return nil, errors.New("cert_data is required for PEM-based TLS source")
		}
		if cfg.KeyData == nil || *cfg.KeyData == "" {
			return nil, errors.New("key_data is required for PEM-based TLS source")
		}
		return slim.TlsSourcePem{
			Cert: *cfg.CertData,
			Key:  *cfg.KeyData,
		}, nil
	}

	return nil, errors.New("either file or PEM source must be specified for TLS certificate")
}

// toSlimKeepaliveConfig converts KeepaliveConfig to *slim.KeepaliveConfig
func (cfg *KeepaliveConfig) toSlimKeepaliveConfig() *slim.KeepaliveConfig {
	return &slim.KeepaliveConfig{
		TcpKeepalive:       cfg.TcpKeepalive,
		Http2Keepalive:     cfg.Http2Keepalive,
		Timeout:            cfg.Timeout,
		KeepAliveWhileIdle: cfg.KeepAliveWhileIdle,
	}
}

// toSlimProxyConfig converts ProxyConfig to slim.ProxyConfig
func (cfg *ProxyConfig) toSlimProxyConfig() (slim.ProxyConfig, error) {
	proxyCfg := slim.ProxyConfig{
		Url:      cfg.Url,
		Username: cfg.Username,
		Password: cfg.Password,
		Headers:  cfg.Headers,
	}

	if cfg.Tls != nil {
		tlsCfg, err := cfg.Tls.toSlimTLSConfig()
		if err != nil {
			return proxyCfg, fmt.Errorf("failed to convert proxy TLS config: %w", err)
		}
		proxyCfg.Tls = tlsCfg
	}

	return proxyCfg, nil
}

// toSlimAuthConfig converts AuthConfig to slim.ClientAuthenticationConfig interface
func (cfg *AuthConfig) toSlimAuthConfig() (slim.ClientAuthenticationConfig, error) {
	switch cfg.Type {
	case "basic":
		if cfg.Basic == nil {
			return nil, errors.New("basic auth configuration is required")
		}
		return slim.ClientAuthenticationConfigBasic{
			Config: slim.BasicAuth{
				Username: cfg.Basic.Username,
				Password: cfg.Basic.Password,
			},
		}, nil

	case "static_jwt":
		if cfg.StaticJwt == nil {
			return nil, errors.New("static JWT configuration is required")
		}
		return slim.ClientAuthenticationConfigStaticJwt{
			Config: slim.StaticJwtAuth{
				TokenFile: cfg.StaticJwt.TokenFile,
				Duration:  cfg.StaticJwt.Duration,
			},
		}, nil

	case "jwt":
		if cfg.Jwt == nil {
			return nil, errors.New("JWT configuration is required")
		}
		if cfg.Jwt.Key == nil {
			return nil, errors.New("JWT key configuration is required for jwt auth")
		}

		// Parse JWT algorithm
		algorithm, err := parseJWTAlgorithm(cfg.Jwt.Key.Algorithm)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT algorithm: %w", err)
		}

		// Parse JWT key format
		format, err := parseJWTKeyFormat(cfg.Jwt.Key.Format)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT key format: %w", err)
		}

		// Parse JWT key data
		keyData, err := cfg.Jwt.Key.Key.toSlimJWTKeyData()
		if err != nil {
			return nil, fmt.Errorf("invalid JWT key data: %w", err)
		}

		// Parse JWT key type (with JwtKeyConfig inside) - always use encoding
		keyType, err := parseJWTKeyType("encoding", algorithm, format, keyData)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT key type: %w", err)
		}

		clientJwtAuth := slim.ClientJwtAuth{
			Duration: cfg.Jwt.Duration,
			Key:      keyType,
		}

		// Add claims if provided
		if len(cfg.Jwt.Audience) > 0 {
			clientJwtAuth.Audience = &cfg.Jwt.Audience
		}
		if cfg.Jwt.Issuer != "" {
			clientJwtAuth.Issuer = &cfg.Jwt.Issuer
		}
		if cfg.Jwt.Subject != "" {
			clientJwtAuth.Subject = &cfg.Jwt.Subject
		}

		return slim.ClientAuthenticationConfigJwt{
			Config: clientJwtAuth,
		}, nil

	case "none", "":
		return slim.ClientAuthenticationConfigNone{}, nil

	default:
		return nil, fmt.Errorf("unknown authentication type: %s", cfg.Type)
	}
}

// parseJWTAlgorithm converts string algorithm to slim.JwtAlgorithm
func parseJWTAlgorithm(algorithm string) (slim.JwtAlgorithm, error) {
	switch algorithm {
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
		return 0, fmt.Errorf("unknown JWT algorithm: %s", algorithm)
	}
}

// parseJWTKeyFormat converts string format to slim.JwtKeyFormat
func parseJWTKeyFormat(format string) (slim.JwtKeyFormat, error) {
	switch format {
	case "pem":
		return slim.JwtKeyFormatPem, nil
	case "jwk":
		return slim.JwtKeyFormatJwk, nil
	case "jwks":
		return slim.JwtKeyFormatJwks, nil
	default:
		return 0, fmt.Errorf("unknown JWT key format: %s", format)
	}
}

// parseJWTKeyType converts string key type and key config to slim.JwtKeyType interface
func parseJWTKeyType(keyType string, algorithm slim.JwtAlgorithm, format slim.JwtKeyFormat, keyData slim.JwtKeyData) (slim.JwtKeyType, error) {
	jwtKeyConfig := slim.JwtKeyConfig{
		Algorithm: algorithm,
		Format:    format,
		Key:       keyData,
	}

	switch keyType {
	case "encoding":
		return slim.JwtKeyTypeEncoding{Key: jwtKeyConfig}, nil
	case "decoding":
		return slim.JwtKeyTypeDecoding{Key: jwtKeyConfig}, nil
	case "autoresolve":
		return slim.JwtKeyTypeAutoresolve{}, nil
	default:
		return nil, fmt.Errorf("unknown JWT key type: %s", keyType)
	}
}

// toSlimJWTKeyData converts JWTKeySource to slim.JwtKeyData interface
func (cfg *JWTKeySource) toSlimJWTKeyData() (slim.JwtKeyData, error) {
	if cfg.File != "" && cfg.Data != "" {
		return nil, errors.New("only one of 'file' or 'data' can be specified for JWT key")
	}

	if cfg.File != "" {
		return slim.JwtKeyDataFile{Path: cfg.File}, nil
	}

	if cfg.Data != "" {
		return slim.JwtKeyDataData{Value: cfg.Data}, nil
	}

	return nil, errors.New("either 'file' or 'data' must be specified for JWT key")
}

// toSlimBackoffConfig converts BackoffConfig to slim.BackoffConfig interface
func (cfg *BackoffConfig) toSlimBackoffConfig() (slim.BackoffConfig, error) {
	switch cfg.Type {
	case "exponential":
		if cfg.Exponential == nil {
			return nil, errors.New("exponential backoff configuration is required")
		}
		if cfg.Exponential.Base == 0 {
			return nil, errors.New("base duration is required for exponential backoff")
		}
		return slim.BackoffConfigExponential{
			Config: slim.ExponentialBackoff{
				Base:        cfg.Exponential.Base,
				Factor:      cfg.Exponential.Factor,
				MaxDelay:    cfg.Exponential.MaxDelay,
				MaxAttempts: cfg.Exponential.MaxAttempts,
				Jitter:      cfg.Exponential.Jitter,
			},
		}, nil

	case "fixed_interval":
		if cfg.FixedInterval == nil {
			return nil, errors.New("fixed interval backoff configuration is required")
		}
		if cfg.FixedInterval.Interval == 0 {
			return nil, errors.New("interval is required for fixed interval backoff")
		}
		return slim.BackoffConfigFixedInterval{
			Config: slim.FixedIntervalBackoff{
				Interval:    cfg.FixedInterval.Interval,
				MaxAttempts: cfg.FixedInterval.MaxAttempts,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown backoff type: %s", cfg.Type)
	}
}
