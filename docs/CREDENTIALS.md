# Admin Credentials Management

## Local Development

For local development, use the `.env` file:

```bash
# Copy the example file
cp .env.example .env

# Edit .env and set your password
ADMIN_USER=admin
ADMIN_PASS=YourSecurePasswordHere123!
```

The `.env` file is in `.gitignore` and will never be committed to Git.

## Kubernetes Deployment

For Kubernetes deployments, credentials are stored as Kubernetes Secrets and injected as environment variables.

### Option 1: Using the Helper Script (Recommended)

```bash
cd charts/app
chmod +x create-secret.sh
./create-secret.sh admin YourSecurePassword123!
```

### Option 2: Manual Secret Creation

```bash
kubectl create secret generic homelab-admin-creds \
  --from-literal=ADMIN_USER=admin \
  --from-literal=ADMIN_PASS=YourSecurePassword123!
```

### Option 3: Using a YAML file (Keep this file secure!)

Create `admin-secret.yaml` (DO NOT commit this to Git):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: homelab-admin-creds
type: Opaque
stringData:
  ADMIN_USER: admin
  ADMIN_PASS: YourSecurePassword123!
```

Apply it:
```bash
kubectl apply -f admin-secret.yaml
```

### Verify the Secret

```bash
# List secrets
kubectl get secrets

# View secret details (base64 encoded)
kubectl describe secret homelab-admin-creds

# Decode and view the password (be careful with this!)
kubectl get secret homelab-admin-creds -o jsonpath='{.data.ADMIN_PASS}' | base64 --decode
```

### Update the Secret

```bash
# Delete and recreate
kubectl delete secret homelab-admin-creds
./create-secret.sh admin NewPassword123!

# Or patch it
kubectl patch secret homelab-admin-creds -p '{"stringData":{"ADMIN_PASS":"NewPassword123!"}}'
```

## How It Works

1. **Local Development**: App reads from `.env` file using `godotenv`
2. **Kubernetes**: App reads from environment variables injected from the secret
3. **Fallback**: If neither exists, defaults to `admin/changeme` (not recommended for production!)

## Security Best Practices

✅ **DO:**
- Use strong, unique passwords
- Store secrets in Kubernetes Secrets
- Use RBAC to restrict who can read secrets
- Rotate passwords periodically
- Consider using external secret managers (HashiCorp Vault, AWS Secrets Manager, etc.)

❌ **DON'T:**
- Commit `.env` files to Git
- Hardcode passwords in code or config files
- Share passwords via insecure channels
- Use default passwords in production
