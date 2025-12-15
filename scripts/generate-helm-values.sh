#!/bin/bash
# Generate Helm values file with embedded data from data/ directory

cat > /tmp/helm-data-values.yaml <<'EOF'
initialData:
  posts:
EOF

# Indent posts.yaml content by 4 spaces
cat data/posts.yaml | sed 's/^/    /' >> /tmp/helm-data-values.yaml

cat >> /tmp/helm-data-values.yaml <<'EOF'
  services:
EOF

# Indent services.yaml content by 4 spaces
cat data/services.yaml | sed 's/^/    /' >> /tmp/helm-data-values.yaml

echo "Generated /tmp/helm-data-values.yaml with embedded data"
echo "Use with: helm upgrade --install ... -f /tmp/helm-data-values.yaml"
