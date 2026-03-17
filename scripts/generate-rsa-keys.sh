#!/bin/bash

set -e

echo "Generating RSA keys..."

mkdir -p config/keys

cd config/keys

echo "Generating access private key..."
openssl genrsa -out accessPrivate.pem 2048

echo "Generating access public key..."
openssl rsa -in accessPrivate.pem -pubout -out accessPublic.pem
chmod 600 accessPrivate.pem
chmod 644 accessPublic.pem

echo "Generating refresh private key..."
openssl genrsa -out refreshPrivate.pem 2048

echo "Generating refresh public key..."
openssl rsa -in refreshPrivate.pem -pubout -out refreshPublic.pem
chmod 600 refreshPrivate.pem
chmod 644 refreshPublic.pem

echo "✅ RSA keys generated successfully!"
echo "📁 Private access key: config/keys/accessPrivate.pem"
echo "📁 Public access key:  config/keys/accessPublic.pem"
echo "📁 Private refresh key: config/keys/refreshPrivate.pem"
echo "📁 Public refresh key:  config/keys/refreshPublic.pem"
echo "⚠️  Keep private.pem secure! Do not commit to repository!"
