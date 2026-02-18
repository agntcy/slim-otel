// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimcommon

import (
	"testing"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions for creating pointer values
func strPtr(s string) *string {
	return &s
}

func TestConnectionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnectionConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid http insecure config",
			config: ConnectionConfig{
				Address: "http://localhost:8080",
				TLS: &TLSConfig{
					Insecure: true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid https secure config",
			config: ConnectionConfig{
				Address: "https://localhost:8443",
				TLS: &TLSConfig{
					Insecure: false,
				},
			},
			wantErr: false,
		},
		{
			name: "missing address",
			config: ConnectionConfig{
				TLS: &TLSConfig{
					Insecure: true,
				},
			},
			wantErr: true,
			errMsg:  "connection address is required",
		},
		{
			name: "http with secure TLS",
			config: ConnectionConfig{
				Address: "http://localhost:8080",
				TLS: &TLSConfig{
					Insecure: false,
				},
			},
			wantErr: true,
			errMsg:  "address must start with https:// for secure TLS config",
		},
		{
			name: "https with insecure TLS",
			config: ConnectionConfig{
				Address: "https://localhost:8443",
				TLS: &TLSConfig{
					Insecure: true,
				},
			},
			wantErr: true,
			errMsg:  "address must start with http:// for insecure TLS config",
		},
		{
			name: "no TLS config with http address",
			config: ConnectionConfig{
				Address: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "no TLS config with https address",
			config: ConnectionConfig{
				Address: "https://localhost:8443",
			},
			wantErr: true,
			errMsg:  "address must start with http:// for insecure connection (no TLS config provided)",
		},
		{
			name: "invalid compression type",
			config: ConnectionConfig{
				Address:     "http://localhost:8080",
				Compression: strPtr("invalid"),
				TLS: &TLSConfig{
					Insecure: true,
				},
			},
			wantErr: true,
			errMsg:  "invalid compression type",
		},
		{
			name: "valid compression type",
			config: ConnectionConfig{
				Address:     "http://localhost:8080",
				Compression: strPtr("gzip"),
				TLS: &TLSConfig{
					Insecure: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTLSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TLSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid insecure config",
			config: TLSConfig{
				Insecure: true,
			},
			wantErr: false,
		},
		{
			name: "valid secure config with CA path",
			config: TLSConfig{
				Insecure: false,
				CASource: &TLSCAConfig{
					Path: strPtr("/path/to/ca.crt"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid secure config with CA data",
			config: TLSConfig{
				Insecure: false,
				CASource: &TLSCAConfig{
					Data: strPtr("PEM data"),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid TLS version",
			config: TLSConfig{
				Insecure:   false,
				TLSVersion: "tls1.1",
			},
			wantErr: true,
			errMsg:  "invalid TLS version",
		},
		{
			name: "valid TLS 1.2",
			config: TLSConfig{
				Insecure:   false,
				TLSVersion: "tls1.2",
			},
			wantErr: false,
		},
		{
			name: "valid TLS 1.3",
			config: TLSConfig{
				Insecure:   false,
				TLSVersion: "tls1.3",
			},
			wantErr: false,
		},
		{
			name: "CA path and data both specified",
			config: TLSConfig{
				Insecure: false,
				CASource: &TLSCAConfig{
					Path: strPtr("/path/to/ca.crt"),
					Data: strPtr("PEM data"),
				},
			},
			wantErr: true,
			errMsg:  "CA path and data cannot both be specified",
		},
		{
			name: "valid mTLS with file source",
			config: TLSConfig{
				Insecure: false,
				Source: &TLSCertKeySource{
					CertFile: strPtr("/path/to/cert.pem"),
					KeyFile:  strPtr("/path/to/key.pem"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid mTLS with PEM source",
			config: TLSConfig{
				Insecure: false,
				Source: &TLSCertKeySource{
					CertData: strPtr("CERT PEM"),
					KeyData:  strPtr("KEY PEM"),
				},
			},
			wantErr: false,
		},
		{
			name: "mTLS with both file and PEM source",
			config: TLSConfig{
				Insecure: false,
				Source: &TLSCertKeySource{
					CertFile: strPtr("/path/to/cert.pem"),
					CertData: strPtr("CERT PEM"),
				},
			},
			wantErr: true,
			errMsg:  "cannot specify both file and PEM sources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTLSConfig(&tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AuthConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid basic auth",
			config: AuthConfig{
				Type: "basic",
				Basic: &BasicAuthConfig{
					Username: "user",
					Password: "pass",
				},
			},
			wantErr: false,
		},
		{
			name: "basic auth missing username",
			config: AuthConfig{
				Type: "basic",
				Basic: &BasicAuthConfig{
					Password: "pass",
				},
			},
			wantErr: true,
			errMsg:  "username is required",
		},
		{
			name: "basic auth missing password",
			config: AuthConfig{
				Type: "basic",
				Basic: &BasicAuthConfig{
					Username: "user",
				},
			},
			wantErr: true,
			errMsg:  "password is required",
		},
		{
			name: "basic auth missing config",
			config: AuthConfig{
				Type: "basic",
			},
			wantErr: true,
			errMsg:  "basic auth configuration is required",
		},
		{
			name: "valid static JWT",
			config: AuthConfig{
				Type: "static_jwt",
				StaticJwt: &StaticJwtAuthConfig{
					TokenFile: "/path/to/token",
					Duration:  5 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "static JWT missing token file",
			config: AuthConfig{
				Type: "static_jwt",
				StaticJwt: &StaticJwtAuthConfig{
					Duration: 5 * time.Minute,
				},
			},
			wantErr: true,
			errMsg:  "token file is required",
		},
		{
			name: "valid JWT with file key",
			config: AuthConfig{
				Type: "jwt",
				Jwt: &JwtAuthConfig{
					Duration: 5 * time.Minute,
					Audience: []string{"audience"},
					Key: &JWTKeyConfig{
						Algorithm: "RS256",
						Format:    "pem",
						Key: &JWTKeySource{
							File: "/path/to/key",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid JWT with data key",
			config: AuthConfig{
				Type: "jwt",
				Jwt: &JwtAuthConfig{
					Duration: 5 * time.Minute,
					Audience: []string{"audience"},
					Key: &JWTKeyConfig{
						Algorithm: "ES256",
						Format:    "pem",
						Key: &JWTKeySource{
							Data: "key data",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "JWT missing audience",
			config: AuthConfig{
				Type: "jwt",
				Jwt: &JwtAuthConfig{
					Duration: 5 * time.Minute,
					Key: &JWTKeyConfig{
						Algorithm: "RS256",
						Format:    "pem",
						Key: &JWTKeySource{
							File: "/path/to/key",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one audience is required",
		},
		{
			name: "JWT missing key config",
			config: AuthConfig{
				Type: "jwt",
				Jwt: &JwtAuthConfig{
					Duration: 5 * time.Minute,
					Audience: []string{"audience"},
				},
			},
			wantErr: true,
			errMsg:  "JWT key configuration is required",
		},
		{
			name: "JWT with both file and data",
			config: AuthConfig{
				Type: "jwt",
				Jwt: &JwtAuthConfig{
					Duration: 5 * time.Minute,
					Audience: []string{"audience"},
					Key: &JWTKeyConfig{
						Algorithm: "RS256",
						Format:    "pem",
						Key: &JWTKeySource{
							File: "/path/to/key",
							Data: "key data",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "JWT key file and data cannot both be specified",
		},
		{
			name: "none auth type",
			config: AuthConfig{
				Type: "none",
			},
			wantErr: false,
		},
		{
			name: "unsupported auth type",
			config: AuthConfig{
				Type: "oauth2",
			},
			wantErr: true,
			errMsg:  "unsupported auth type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuthConfig(&tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBackoffConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  BackoffConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid exponential backoff",
			config: BackoffConfig{
				Type: "exponential",
				Exponential: &ExponentialBackoffConfig{
					Base:        100 * time.Millisecond,
					Factor:      2,
					MaxDelay:    30 * time.Second,
					MaxAttempts: 5,
					Jitter:      true,
				},
			},
			wantErr: false,
		},
		{
			name: "exponential backoff missing config",
			config: BackoffConfig{
				Type: "exponential",
			},
			wantErr: true,
			errMsg:  "exponential backoff configuration is required",
		},
		{
			name: "exponential backoff missing base",
			config: BackoffConfig{
				Type: "exponential",
				Exponential: &ExponentialBackoffConfig{
					Factor:      2,
					MaxDelay:    30 * time.Second,
					MaxAttempts: 5,
				},
			},
			wantErr: true,
			errMsg:  "base duration is required",
		},
		{
			name: "valid fixed interval backoff",
			config: BackoffConfig{
				Type: "fixed_interval",
				FixedInterval: &FixedIntervalBackoffConfig{
					Interval:    1 * time.Second,
					MaxAttempts: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "fixed interval backoff missing config",
			config: BackoffConfig{
				Type: "fixed_interval",
			},
			wantErr: true,
			errMsg:  "fixed interval backoff configuration is required",
		},
		{
			name: "fixed interval backoff missing interval",
			config: BackoffConfig{
				Type: "fixed_interval",
				FixedInterval: &FixedIntervalBackoffConfig{
					MaxAttempts: 3,
				},
			},
			wantErr: true,
			errMsg:  "interval is required",
		},
		{
			name: "invalid backoff type",
			config: BackoffConfig{
				Type: "linear",
			},
			wantErr: true,
			errMsg:  "invalid backoff type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBackoffConfig(&tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProxyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ProxyConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid proxy with URL",
			config: ProxyConfig{
				Url: strPtr("http://proxy.example.com:8080"),
			},
			wantErr: false,
		},
		{
			name: "valid proxy with auth",
			config: ProxyConfig{
				Url:      strPtr("http://proxy.example.com:8080"),
				Username: strPtr("user"),
				Password: strPtr("pass"),
			},
			wantErr: false,
		},
		{
			name: "proxy auth without URL",
			config: ProxyConfig{
				Username: strPtr("user"),
				Password: strPtr("pass"),
			},
			wantErr: true,
			errMsg:  "proxy URL is required when username or password is specified",
		},
		{
			name: "valid proxy with TLS",
			config: ProxyConfig{
				Url: strPtr("https://proxy.example.com:8443"),
				Tls: &TLSConfig{
					Insecure: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProxyConfig(&tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCompressionValidation(t *testing.T) {
	validTypes := []string{"gzip", "zlib", "deflate", "snappy", "zstd", "lz4", "none", "empty"}
	for _, ct := range validTypes {
		t.Run("valid_"+ct, func(t *testing.T) {
			err := validateCompression(ct)
			assert.NoError(t, err)
		})
	}

	t.Run("invalid compression", func(t *testing.T) {
		err := validateCompression("bzip2")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid compression type")
	})
}

func TestParseCompressionType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slim.CompressionType
		wantErr  bool
	}{
		{"gzip", "gzip", slim.CompressionTypeGzip, false},
		{"zlib", "zlib", slim.CompressionTypeZlib, false},
		{"deflate", "deflate", slim.CompressionTypeDeflate, false},
		{"snappy", "snappy", slim.CompressionTypeSnappy, false},
		{"zstd", "zstd", slim.CompressionTypeZstd, false},
		{"lz4", "lz4", slim.CompressionTypeLz4, false},
		{"none", "none", slim.CompressionTypeNone, false},
		{"empty", "empty", slim.CompressionTypeEmpty, false},
		{"invalid", "bzip2", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCompressionType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseJWTAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slim.JwtAlgorithm
		wantErr  bool
	}{
		{"HS256", "HS256", slim.JwtAlgorithmHs256, false},
		{"HS384", "HS384", slim.JwtAlgorithmHs384, false},
		{"HS512", "HS512", slim.JwtAlgorithmHs512, false},
		{"ES256", "ES256", slim.JwtAlgorithmEs256, false},
		{"ES384", "ES384", slim.JwtAlgorithmEs384, false},
		{"RS256", "RS256", slim.JwtAlgorithmRs256, false},
		{"RS384", "RS384", slim.JwtAlgorithmRs384, false},
		{"RS512", "RS512", slim.JwtAlgorithmRs512, false},
		{"PS256", "PS256", slim.JwtAlgorithmPs256, false},
		{"PS384", "PS384", slim.JwtAlgorithmPs384, false},
		{"PS512", "PS512", slim.JwtAlgorithmPs512, false},
		{"EdDSA", "EdDSA", slim.JwtAlgorithmEdDsa, false},
		{"invalid", "INVALID", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJWTAlgorithm(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseJWTKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slim.JwtKeyFormat
		wantErr  bool
	}{
		{"pem", "pem", slim.JwtKeyFormatPem, false},
		{"jwk", "jwk", slim.JwtKeyFormatJwk, false},
		{"jwks", "jwks", slim.JwtKeyFormatJwks, false},
		{"invalid", "pkcs12", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJWTKeyFormat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTLSCAConfig_ToSlimCASource(t *testing.T) {
	tests := []struct {
		name     string
		config   TLSCAConfig
		expected slim.CaSource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "file source",
			config: TLSCAConfig{
				Path: strPtr("/path/to/ca.pem"),
			},
			expected: slim.CaSourceFile{Path: "/path/to/ca.pem"},
			wantErr:  false,
		},
		{
			name: "PEM source",
			config: TLSCAConfig{
				Data: strPtr("PEM DATA"),
			},
			expected: slim.CaSourcePem{Data: "PEM DATA"},
			wantErr:  false,
		},
		{
			name:     "none source",
			config:   TLSCAConfig{},
			expected: slim.CaSourceNone{},
			wantErr:  false,
		},
		{
			name: "both path and data",
			config: TLSCAConfig{
				Path: strPtr("/path/to/ca.pem"),
				Data: strPtr("PEM DATA"),
			},
			wantErr: true,
			errMsg:  "CA path and data cannot both be specified",
		},
		{
			name: "empty path",
			config: TLSCAConfig{
				Path: strPtr(""),
			},
			wantErr: true,
			errMsg:  "CA file path cannot be empty",
		},
		{
			name: "empty data",
			config: TLSCAConfig{
				Data: strPtr(""),
			},
			wantErr: true,
			errMsg:  "CA PEM data cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.toSlimCASource()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTLSCertKeySource_ToSlimTLSSource(t *testing.T) {
	tests := []struct {
		name     string
		config   TLSCertKeySource
		expected slim.TlsSource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "file source",
			config: TLSCertKeySource{
				CertFile: strPtr("/path/to/cert.pem"),
				KeyFile:  strPtr("/path/to/key.pem"),
			},
			expected: slim.TlsSourceFile{
				Cert: "/path/to/cert.pem",
				Key:  "/path/to/key.pem",
			},
			wantErr: false,
		},
		{
			name: "PEM source",
			config: TLSCertKeySource{
				CertData: strPtr("CERT PEM"),
				KeyData:  strPtr("KEY PEM"),
			},
			expected: slim.TlsSourcePem{
				Cert: "CERT PEM",
				Key:  "KEY PEM",
			},
			wantErr: false,
		},
		{
			name: "both file and PEM",
			config: TLSCertKeySource{
				CertFile: strPtr("/path/to/cert.pem"),
				CertData: strPtr("CERT PEM"),
			},
			wantErr: true,
			errMsg:  "cannot specify both file and PEM sources",
		},
		{
			name: "missing cert file",
			config: TLSCertKeySource{
				KeyFile: strPtr("/path/to/key.pem"),
			},
			wantErr: true,
			errMsg:  "cert_file is required",
		},
		{
			name: "missing key file",
			config: TLSCertKeySource{
				CertFile: strPtr("/path/to/cert.pem"),
			},
			wantErr: true,
			errMsg:  "key_file is required",
		},
		{
			name: "missing cert data",
			config: TLSCertKeySource{
				KeyData: strPtr("KEY PEM"),
			},
			wantErr: true,
			errMsg:  "cert_data is required",
		},
		{
			name: "missing key data",
			config: TLSCertKeySource{
				CertData: strPtr("CERT PEM"),
			},
			wantErr: true,
			errMsg:  "key_data is required",
		},
		{
			name:    "no source specified",
			config:  TLSCertKeySource{},
			wantErr: true,
			errMsg:  "either file or PEM source must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.toSlimTLSSource()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestJWTKeySource_ToSlimJWTKeyData(t *testing.T) {
	tests := []struct {
		name     string
		config   JWTKeySource
		expected slim.JwtKeyData
		wantErr  bool
		errMsg   string
	}{
		{
			name: "file source",
			config: JWTKeySource{
				File: "/path/to/key.pem",
			},
			expected: slim.JwtKeyDataFile{Path: "/path/to/key.pem"},
			wantErr:  false,
		},
		{
			name: "data source",
			config: JWTKeySource{
				Data: "KEY DATA",
			},
			expected: slim.JwtKeyDataData{Value: "KEY DATA"},
			wantErr:  false,
		},
		{
			name: "both file and data",
			config: JWTKeySource{
				File: "/path/to/key.pem",
				Data: "KEY DATA",
			},
			wantErr: true,
			errMsg:  "only one of 'file' or 'data' can be specified",
		},
		{
			name:    "neither file nor data",
			config:  JWTKeySource{},
			wantErr: true,
			errMsg:  "either 'file' or 'data' must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.toSlimJWTKeyData()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConnectionConfig_ToSlimClientConfig(t *testing.T) {
	t.Run("basic http config", func(t *testing.T) {
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			TLS: &TLSConfig{
				Insecure: true,
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080", clientCfg.Endpoint)
		assert.NotNil(t, clientCfg.Tls)
		assert.True(t, clientCfg.Tls.Insecure)
	})

	t.Run("config with headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Custom-Header": "value",
		}
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			Headers: &headers,
			TLS: &TLSConfig{
				Insecure: true,
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.Equal(t, headers, clientCfg.Headers)
	})

	t.Run("config with compression", func(t *testing.T) {
		compression := "gzip"
		config := ConnectionConfig{
			Address:     "http://localhost:8080",
			Compression: &compression,
			TLS: &TLSConfig{
				Insecure: true,
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.NotNil(t, clientCfg.Compression)
		assert.Equal(t, slim.CompressionTypeGzip, *clientCfg.Compression)
	})

	t.Run("config with keepalive", func(t *testing.T) {
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			TLS: &TLSConfig{
				Insecure: true,
			},
			Keepalive: &KeepaliveConfig{
				TcpKeepalive:       30 * time.Second,
				Http2Keepalive:     60 * time.Second,
				Timeout:            10 * time.Second,
				KeepAliveWhileIdle: true,
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.NotNil(t, clientCfg.Keepalive)
		assert.Equal(t, 30*time.Second, clientCfg.Keepalive.TcpKeepalive)
		assert.Equal(t, 60*time.Second, clientCfg.Keepalive.Http2Keepalive)
	})

	t.Run("config with basic auth", func(t *testing.T) {
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			TLS: &TLSConfig{
				Insecure: true,
			},
			Auth: &AuthConfig{
				Type: "basic",
				Basic: &BasicAuthConfig{
					Username: "user",
					Password: "pass",
				},
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.NotNil(t, clientCfg.Auth)
		basicAuth, ok := clientCfg.Auth.(slim.ClientAuthenticationConfigBasic)
		assert.True(t, ok)
		assert.Equal(t, "user", basicAuth.Config.Username)
		assert.Equal(t, "pass", basicAuth.Config.Password)
	})

	t.Run("config with exponential backoff", func(t *testing.T) {
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			TLS: &TLSConfig{
				Insecure: true,
			},
			Backoff: &BackoffConfig{
				Type: "exponential",
				Exponential: &ExponentialBackoffConfig{
					Base:        100 * time.Millisecond,
					Factor:      2,
					MaxDelay:    30 * time.Second,
					MaxAttempts: 5,
					Jitter:      true,
				},
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.NotNil(t, clientCfg.Backoff)
		expBackoff, ok := clientCfg.Backoff.(slim.BackoffConfigExponential)
		assert.True(t, ok)
		assert.Equal(t, 100*time.Millisecond, expBackoff.Config.Base)
		assert.Equal(t, uint64(2), expBackoff.Config.Factor)
		assert.Equal(t, 30*time.Second, expBackoff.Config.MaxDelay)
		assert.Equal(t, uint64(5), expBackoff.Config.MaxAttempts)
		assert.True(t, expBackoff.Config.Jitter)
	})

	t.Run("config with fixed interval backoff", func(t *testing.T) {
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			TLS: &TLSConfig{
				Insecure: true,
			},
			Backoff: &BackoffConfig{
				Type: "fixed_interval",
				FixedInterval: &FixedIntervalBackoffConfig{
					Interval:    1 * time.Second,
					MaxAttempts: 3,
				},
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.NotNil(t, clientCfg.Backoff)
		fixedBackoff, ok := clientCfg.Backoff.(slim.BackoffConfigFixedInterval)
		assert.True(t, ok)
		assert.Equal(t, 1*time.Second, fixedBackoff.Config.Interval)
		assert.Equal(t, uint64(3), fixedBackoff.Config.MaxAttempts)
	})

	t.Run("config with default values", func(t *testing.T) {
		config := ConnectionConfig{
			Address: "http://localhost:8080",
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)

		// Should have default TLS config
		assert.NotNil(t, clientCfg.Tls)
		assert.True(t, clientCfg.Tls.Insecure)

		// Should have default auth (none)
		_, ok := clientCfg.Auth.(slim.ClientAuthenticationConfigNone)
		assert.True(t, ok)

		// Should have default backoff (exponential)
		expBackoff, ok := clientCfg.Backoff.(slim.BackoffConfigExponential)
		assert.True(t, ok)
		assert.Equal(t, 100*time.Millisecond, expBackoff.Config.Base)

		// Should have default proxy config
		assert.NotNil(t, clientCfg.Proxy)
		assert.Nil(t, clientCfg.Proxy.Url)

		// Should have empty headers map
		assert.NotNil(t, clientCfg.Headers)
		assert.Equal(t, 0, len(clientCfg.Headers))
	})

	t.Run("config with proxy", func(t *testing.T) {
		proxyUrl := "http://proxy.example.com:8080"
		config := ConnectionConfig{
			Address: "http://localhost:8080",
			TLS: &TLSConfig{
				Insecure: true,
			},
			Proxy: &ProxyConfig{
				Url:      &proxyUrl,
				Username: strPtr("proxyuser"),
				Password: strPtr("proxypass"),
			},
		}

		clientCfg, err := config.ToSlimClientConfig()
		require.NoError(t, err)
		assert.NotNil(t, clientCfg.Proxy)
		assert.NotNil(t, clientCfg.Proxy.Url)
		assert.Equal(t, proxyUrl, *clientCfg.Proxy.Url)
		assert.NotNil(t, clientCfg.Proxy.Username)
		assert.Equal(t, "proxyuser", *clientCfg.Proxy.Username)
	})
}

func TestAuthConfig_ToSlimAuthConfig(t *testing.T) {
	t.Run("basic auth", func(t *testing.T) {
		config := AuthConfig{
			Type: "basic",
			Basic: &BasicAuthConfig{
				Username: "testuser",
				Password: "testpass",
			},
		}

		result, err := config.toSlimAuthConfig()
		require.NoError(t, err)
		basicAuth, ok := result.(slim.ClientAuthenticationConfigBasic)
		assert.True(t, ok)
		assert.Equal(t, "testuser", basicAuth.Config.Username)
		assert.Equal(t, "testpass", basicAuth.Config.Password)
	})

	t.Run("static JWT", func(t *testing.T) {
		config := AuthConfig{
			Type: "static_jwt",
			StaticJwt: &StaticJwtAuthConfig{
				TokenFile: "/path/to/token",
				Duration:  5 * time.Minute,
			},
		}

		result, err := config.toSlimAuthConfig()
		require.NoError(t, err)
		staticJwt, ok := result.(slim.ClientAuthenticationConfigStaticJwt)
		assert.True(t, ok)
		assert.Equal(t, "/path/to/token", staticJwt.Config.TokenFile)
		assert.Equal(t, 5*time.Minute, staticJwt.Config.Duration)
	})

	t.Run("JWT with file key", func(t *testing.T) {
		config := AuthConfig{
			Type: "jwt",
			Jwt: &JwtAuthConfig{
				Duration: 10 * time.Minute,
				Audience: []string{"aud1", "aud2"},
				Issuer:   "test-issuer",
				Subject:  "test-subject",
				Key: &JWTKeyConfig{
					Algorithm: "RS256",
					Format:    "pem",
					Key: &JWTKeySource{
						File: "/path/to/key.pem",
					},
				},
			},
		}

		result, err := config.toSlimAuthConfig()
		require.NoError(t, err)
		jwtAuth, ok := result.(slim.ClientAuthenticationConfigJwt)
		assert.True(t, ok)
		assert.Equal(t, 10*time.Minute, jwtAuth.Config.Duration)
		assert.NotNil(t, jwtAuth.Config.Audience)
		assert.Equal(t, []string{"aud1", "aud2"}, *jwtAuth.Config.Audience)
		assert.NotNil(t, jwtAuth.Config.Issuer)
		assert.Equal(t, "test-issuer", *jwtAuth.Config.Issuer)
		assert.NotNil(t, jwtAuth.Config.Subject)
		assert.Equal(t, "test-subject", *jwtAuth.Config.Subject)
	})

	t.Run("none auth", func(t *testing.T) {
		config := AuthConfig{
			Type: "none",
		}

		result, err := config.toSlimAuthConfig()
		require.NoError(t, err)
		_, ok := result.(slim.ClientAuthenticationConfigNone)
		assert.True(t, ok)
	})

	t.Run("empty auth type defaults to none", func(t *testing.T) {
		config := AuthConfig{
			Type: "",
		}

		result, err := config.toSlimAuthConfig()
		require.NoError(t, err)
		_, ok := result.(slim.ClientAuthenticationConfigNone)
		assert.True(t, ok)
	})
}

func TestBackoffConfig_ToSlimBackoffConfig(t *testing.T) {
	t.Run("exponential backoff", func(t *testing.T) {
		config := BackoffConfig{
			Type: "exponential",
			Exponential: &ExponentialBackoffConfig{
				Base:        200 * time.Millisecond,
				Factor:      3,
				MaxDelay:    60 * time.Second,
				MaxAttempts: 10,
				Jitter:      false,
			},
		}

		result, err := config.toSlimBackoffConfig()
		require.NoError(t, err)
		expBackoff, ok := result.(slim.BackoffConfigExponential)
		assert.True(t, ok)
		assert.Equal(t, 200*time.Millisecond, expBackoff.Config.Base)
		assert.Equal(t, uint64(3), expBackoff.Config.Factor)
		assert.Equal(t, 60*time.Second, expBackoff.Config.MaxDelay)
		assert.Equal(t, uint64(10), expBackoff.Config.MaxAttempts)
		assert.False(t, expBackoff.Config.Jitter)
	})

	t.Run("fixed interval backoff", func(t *testing.T) {
		config := BackoffConfig{
			Type: "fixed_interval",
			FixedInterval: &FixedIntervalBackoffConfig{
				Interval:    2 * time.Second,
				MaxAttempts: 5,
			},
		}

		result, err := config.toSlimBackoffConfig()
		require.NoError(t, err)
		fixedBackoff, ok := result.(slim.BackoffConfigFixedInterval)
		assert.True(t, ok)
		assert.Equal(t, 2*time.Second, fixedBackoff.Config.Interval)
		assert.Equal(t, uint64(5), fixedBackoff.Config.MaxAttempts)
	})
}

func TestTLSConfig_ToSlimTLSConfig(t *testing.T) {
	t.Run("insecure config", func(t *testing.T) {
		config := TLSConfig{
			Insecure: true,
		}

		result, err := config.toSlimTLSConfig()
		require.NoError(t, err)
		assert.True(t, result.Insecure)
		assert.Equal(t, "tls1.3", result.TlsVersion)
		assert.True(t, result.IncludeSystemCaCertsPool)
	})

	t.Run("secure config with CA file", func(t *testing.T) {
		includeSystemCA := false
		config := TLSConfig{
			Insecure:                 false,
			TLSVersion:               "tls1.2",
			IncludeSystemCACertsPool: &includeSystemCA,
			CASource: &TLSCAConfig{
				Path: strPtr("/path/to/ca.crt"),
			},
		}

		result, err := config.toSlimTLSConfig()
		require.NoError(t, err)
		assert.False(t, result.Insecure)
		assert.Equal(t, "tls1.2", result.TlsVersion)
		assert.False(t, result.IncludeSystemCaCertsPool)
		caFile, ok := result.CaSource.(slim.CaSourceFile)
		assert.True(t, ok)
		assert.Equal(t, "/path/to/ca.crt", caFile.Path)
	})

	t.Run("config with mTLS file source", func(t *testing.T) {
		config := TLSConfig{
			Insecure: false,
			Source: &TLSCertKeySource{
				CertFile: strPtr("/path/to/cert.pem"),
				KeyFile:  strPtr("/path/to/key.pem"),
			},
		}

		result, err := config.toSlimTLSConfig()
		require.NoError(t, err)
		tlsFile, ok := result.Source.(slim.TlsSourceFile)
		assert.True(t, ok)
		assert.Equal(t, "/path/to/cert.pem", tlsFile.Cert)
		assert.Equal(t, "/path/to/key.pem", tlsFile.Key)
	})

	t.Run("config with mTLS PEM source", func(t *testing.T) {
		config := TLSConfig{
			Insecure: false,
			Source: &TLSCertKeySource{
				CertData: strPtr("CERT PEM DATA"),
				KeyData:  strPtr("KEY PEM DATA"),
			},
		}

		result, err := config.toSlimTLSConfig()
		require.NoError(t, err)
		tlsPem, ok := result.Source.(slim.TlsSourcePem)
		assert.True(t, ok)
		assert.Equal(t, "CERT PEM DATA", tlsPem.Cert)
		assert.Equal(t, "KEY PEM DATA", tlsPem.Key)
	})

	t.Run("config without source defaults to TlsSourceNone", func(t *testing.T) {
		config := TLSConfig{
			Insecure: false,
		}

		result, err := config.toSlimTLSConfig()
		require.NoError(t, err)
		_, ok := result.Source.(slim.TlsSourceNone)
		assert.True(t, ok)
	})
}

func TestProxyConfig_ToSlimProxyConfig(t *testing.T) {
	t.Run("basic proxy config", func(t *testing.T) {
		url := "http://proxy.example.com:8080"
		config := ProxyConfig{
			Url: &url,
		}

		result, err := config.toSlimProxyConfig()
		require.NoError(t, err)
		assert.NotNil(t, result.Url)
		assert.Equal(t, url, *result.Url)
	})

	t.Run("proxy with auth", func(t *testing.T) {
		url := "http://proxy.example.com:8080"
		config := ProxyConfig{
			Url:      &url,
			Username: strPtr("proxyuser"),
			Password: strPtr("proxypass"),
		}

		result, err := config.toSlimProxyConfig()
		require.NoError(t, err)
		assert.NotNil(t, result.Username)
		assert.Equal(t, "proxyuser", *result.Username)
		assert.NotNil(t, result.Password)
		assert.Equal(t, "proxypass", *result.Password)
	})

	t.Run("proxy with TLS", func(t *testing.T) {
		url := "https://proxy.example.com:8443"
		config := ProxyConfig{
			Url: &url,
			Tls: &TLSConfig{
				Insecure:   false,
				TLSVersion: "tls1.3",
			},
		}

		result, err := config.toSlimProxyConfig()
		require.NoError(t, err)
		assert.False(t, result.Tls.Insecure)
		assert.Equal(t, "tls1.3", result.Tls.TlsVersion)
	})
}

func TestKeepaliveConfig_ToSlimKeepaliveConfig(t *testing.T) {
	config := KeepaliveConfig{
		TcpKeepalive:       30 * time.Second,
		Http2Keepalive:     60 * time.Second,
		Timeout:            10 * time.Second,
		KeepAliveWhileIdle: true,
	}

	result := config.toSlimKeepaliveConfig()
	assert.NotNil(t, result)
	assert.Equal(t, 30*time.Second, result.TcpKeepalive)
	assert.Equal(t, 60*time.Second, result.Http2Keepalive)
	assert.Equal(t, 10*time.Second, result.Timeout)
	assert.True(t, result.KeepAliveWhileIdle)
}
