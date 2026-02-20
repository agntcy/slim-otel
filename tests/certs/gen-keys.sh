#!/bin/bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0


CA_CNF="ca.cnf"

SLIM_NODE_CNF="slim-node.cnf"
SLIM_NODE_CA_KEY="slim-node-ca-key.pem"
SLIM_NODE_CA_CERT="slim-node-ca-cert.pem"
SLIM_NODE_CSR="slim-node-csr.pem"
SLIM_NODE_CERT="slim-node-cert.pem"
SLIM_NODE_KEY="slim-node-key.pem"

EXPORTER_CNF="exporter.cnf"
EXPORTER_CA_KEY="exporter-ca-key.pem"
EXPORTER_CA_CERT="exporter-ca-cert.pem"
EXPORTER_CSR="exporter-csr.pem"
EXPORTER_CERT="exporter-cert.pem"
EXPORTER_KEY="exporter-key.pem"

RECEIVER_CNF="receiver.cnf"
RECEIVER_CA_KEY="receiver-ca-key.pem"
RECEIVER_CA_CERT="receiver-ca-cert.pem"
RECEIVER_CSR="receiver-csr.pem"
RECEIVER_CERT="receiver-cert.pem"
RECEIVER_KEY="receiver-key.pem"

# SLIM Node CA Key
openssl ecparam             \
    -genkey                 \
    -name secp384r1         \
    -out ${SLIM_NODE_CA_KEY}

# SLIM Node CA Cert
openssl req                 \
    -x509                   \
    -new                    \
    -key ${SLIM_NODE_CA_KEY}   \
    -out ${SLIM_NODE_CA_CERT}  \
    -config ${CA_CNF}       \
    -days 3650

# SLIM Node Key
openssl ecparam     \
    -genkey         \
    -name secp384r1 \
    -out ${SLIM_NODE_KEY}

# SLIM Node CSR
openssl req                 \
    -new                    \
    -key ${SLIM_NODE_KEY}      \
    -out ${SLIM_NODE_CSR}      \
    -config ${SLIM_NODE_CNF}

# SLIM Node Cert
openssl x509                \
    -req                    \
    -in ${SLIM_NODE_CSR}       \
    -CA ${SLIM_NODE_CA_CERT}   \
    -CAkey ${SLIM_NODE_CA_KEY} \
    -CAcreateserial         \
    -out ${SLIM_NODE_CERT}     \
    -extfile ${SLIM_NODE_CNF}  \
    -extensions req_ext     \
    -days 3650

# Exporter CA Key
openssl ecparam             \
    -genkey                 \
    -name secp384r1         \
    -out ${EXPORTER_CA_KEY}

# Exporter CA Cert
openssl req                 \
    -x509                   \
    -new                    \
    -key ${EXPORTER_CA_KEY}   \
    -out ${EXPORTER_CA_CERT}  \
    -config ${CA_CNF}       \
    -days 3650

# Exporter Key
openssl ecparam     \
    -genkey         \
    -name secp384r1 \
    -out ${EXPORTER_KEY}

# Exporter CSR
openssl req                 \
    -new                    \
    -key ${EXPORTER_KEY}      \
    -out ${EXPORTER_CSR}      \
    -config ${EXPORTER_CNF}

# Exporter Cert
openssl x509                \
    -req                    \
    -in ${EXPORTER_CSR}       \
    -CA ${EXPORTER_CA_CERT}   \
    -CAkey ${EXPORTER_CA_KEY} \
    -CAcreateserial         \
    -out ${EXPORTER_CERT}     \
    -extfile ${EXPORTER_CNF}  \
    -extensions req_ext     \
    -days 3650

# Receiver CA Key
openssl ecparam             \
    -genkey                 \
    -name secp384r1         \
    -out ${RECEIVER_CA_KEY}

# Receiver CA Cert
openssl req                 \
    -x509                   \
    -new                    \
    -key ${RECEIVER_CA_KEY}   \
    -out ${RECEIVER_CA_CERT}  \
    -config ${CA_CNF}       \
    -days 3650

# Receiver Key
openssl ecparam     \
    -genkey         \
    -name secp384r1 \
    -out ${RECEIVER_KEY}

# Receiver CSR
openssl req                 \
    -new                    \
    -key ${RECEIVER_KEY}      \
    -out ${RECEIVER_CSR}      \
    -config ${RECEIVER_CNF}

# Receiver Cert
openssl x509                \
    -req                    \
    -in ${RECEIVER_CSR}       \
    -CA ${RECEIVER_CA_CERT}   \
    -CAkey ${RECEIVER_CA_KEY} \
    -CAcreateserial         \
    -out ${RECEIVER_CERT}     \
    -extfile ${RECEIVER_CNF}  \
    -extensions req_ext     \
    -days 3650
