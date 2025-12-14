#!/bin/bash
# Helper script to create Kubernetes secret for admin credentials
# Usage: ./create-secret.sh <username> <password>

ADMIN_USER=${1:-admin}
ADMIN_PASS=${2}

if [ -z "$ADMIN_PASS" ]; then
  echo "Error: Password is required"
  echo "Usage: ./create-secret.sh <username> <password>"
  echo "Example: ./create-secret.sh admin MySecurePassword123!"
  exit 1
fi

echo "Creating Kubernetes secret 'homelab-admin-creds'..."

kubectl create secret generic homelab-admin-creds \
  --from-literal=ADMIN_USER="$ADMIN_USER" \
  --from-literal=ADMIN_PASS="$ADMIN_PASS" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Secret created/updated successfully!"
echo ""
echo "The secret will be automatically injected into the deployment as environment variables."
echo "Deploy with: helm upgrade --install homelabsite ./charts/app"
