#!/bin/bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

# Script to generate JWT keys for different algorithms
# Based on https://github.com/agntcy/slim/tree/main/data-plane/config/crypto

set -e

JWT_DIR="jwt"

# Create jwt directory if it doesn't exist
mkdir -p "$JWT_DIR"

cd "$JWT_DIR"

echo "Generating JWT keys..."

# RSA 2048-bit keys (for RS256, RS384, RS512, PS256, PS384, PS512)
echo "  - RSA keys (rsa.pem, rsa-public.pem)"
openssl genrsa -out rsa.pem 2048
openssl rsa -in rsa.pem -pubout -out rsa-public.pem

# Generate a wrong RSA key for testing
echo "  - RSA wrong key (rsa-wrong.pem)"
openssl genrsa -out rsa-wrong.pem 2048

# EC P-256 keys (for ES256)
echo "  - EC P-256 keys (ec256.pem, ec256-public.pem)"
openssl ecparam -name prime256v1 -genkey -noout -out ec256.pem
openssl ec -in ec256.pem -pubout -out ec256-public.pem

# Generate a wrong EC P-256 key for testing
echo "  - EC P-256 wrong key (ec256-wrong.pem)"
openssl ecparam -name prime256v1 -genkey -noout -out ec256-wrong.pem

# EC P-384 keys (for ES384)
echo "  - EC P-384 keys (ec384.pem, ec384-public.pem)"
openssl ecparam -name secp384r1 -genkey -noout -out ec384.pem
openssl ec -in ec384.pem -pubout -out ec384-public.pem

# Generate a wrong EC P-384 key for testing
echo "  - EC P-384 wrong key (ec384-wrong.pem)"
openssl ecparam -name secp384r1 -genkey -noout -out ec384-wrong.pem

# EdDSA (Ed25519) keys
echo "  - EdDSA keys (eddsa.pem, eddsa-public.pem)"
openssl genpkey -algorithm ED25519 -out eddsa.pem
openssl pkey -in eddsa.pem -pubout -out eddsa-public.pem

# Generate a wrong EdDSA key for testing
echo "  - EdDSA wrong key (eddsa-wrong.pem)"
openssl genpkey -algorithm ED25519 -out eddsa-wrong.pem

cd ..

echo ""
echo "JWT keys generated successfully in $JWT_DIR/"
echo ""
echo "Key files created:"
echo "  RSA (RS*/PS*):  rsa.pem, rsa-public.pem, rsa-wrong.pem"
echo "  EC P-256 (ES256): ec256.pem, ec256-public.pem, ec256-wrong.pem"
echo "  EC P-384 (ES384): ec384.pem, ec384-public.pem, ec384-wrong.pem"
echo "  EdDSA: eddsa.pem, eddsa-public.pem, eddsa-wrong.pem"
echo ""
echo "Note: For HMAC (HS256/HS384/HS512), use a shared secret string (min 32 chars)"
