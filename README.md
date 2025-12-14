# Atarnet Homelab Site

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Go Version](https://img.shields.io/badge/go-1.25-blue)]()

A personal homelab portfolio and blog showcasing my Kubernetes infrastructure, services, and technical journey as a Senior Software Engineer.

**Live Site**: [atarnet.org](https://atarnet.org)

## Purpose

This site serves as:
- **Service Showcase**: Central hub displaying all services running in my Kubernetes homelab
- **Technical Blog**: Deep-dives into Kubernetes, Go, DevOps, infrastructure, and system architecture
- **Documentation**: Reference for my homelab setup, CI/CD pipelines, and deployment patterns
- **Portfolio**: Demonstration of infrastructure engineering and best practices

## Architecture

Built with simplicity and performance in mind:

- **Backend**: Go with embedded templates and static files (embed.FS)
- **Frontend**: HTMX for dynamic interactions without heavy JavaScript frameworks
- **Styling**: Custom CSS with modern dark theme
- **Content**: YAML-based configuration for easy updates
- **Deployment**: Kubernetes with Helm charts, following GitOps principles
- **CI/CD**: Jenkins with Kaniko for containerized builds and Trivy for security scanning
- **External Access**: Cloudflare Tunnels with Zero Trust authentication

### Technology Choices

- **Go**: Single binary deployment, minimal memory footprint (~20MB), fast startup
- **HTMX**: Progressive enhancement, SEO-friendly, minimal JavaScript
- **Embedded Files**: No runtime dependencies, simplified Docker images
- **Kubernetes**: Orchestration matching enterprise patterns
- **ConfigMaps**: Hot-reload content without rebuilding container images

## Running Locally

### Prerequisites

- Go 1.22 or higher
- Git

### Development Setup

```bash
# Clone the repository
git clone https://github.com/tinotenda-alfaneti/homelabsite.git
cd homelabsite

# Install dependencies
go mod download

# Create environment file
cat > .env << EOF
ADMIN_USER=admin
ADMIN_PASS=your_secure_password
PORT=8082
EOF

# Run the application
go run .
```

The application will start on `http://localhost:8082`

**Default Admin Credentials** (if .env not configured):
- Username: `admin`
- Password: `changeme`

### Using Make

```bash
# Build binary
make build

# Run application
make run

# Build Docker image
make docker-build

# Run in Docker
make docker-run

# Clean build artifacts
make clean
```

## Deployment

### Kubernetes with Helm

```bash
# Create namespace
kubectl create namespace homelabsite-ns

# Create secret for admin credentials (IMPORTANT!)
kubectl create secret generic homelab-admin-creds \
  --namespace homelabsite-ns \
  --from-literal=ADMIN_USER=admin \
  --from-literal=ADMIN_PASS=YourSecurePassword123!

# Deploy with Helm
helm upgrade --install homelabsite ./charts/app \
  --namespace homelabsite-ns \
  --set image.repository=tinorodney/homelabsite \
  --set image.tag=v0.0.1 \
  --set-file configMap.files.config\\.yaml=./config/config.yaml
```

See [CREDENTIALS.md](./CREDENTIALS.md) for detailed credential management instructions.

### CI/CD Pipeline

The Jenkinsfile automates the complete deployment workflow:

1. **Build**: Kaniko builds container images inside Kubernetes (no Docker daemon required)
2. **Scan**: Trivy scans for vulnerabilities and misconfigurations
3. **Push**: Clean images pushed to Docker Hub registry
4. **Deploy**: Helm deploys updated charts to Kubernetes
5. **Verify**: Health checks confirm successful deployment

Simply push to GitHub and Jenkins handles the rest.

## Content Management

## Content Management

### Admin Panel

The web-based admin interface provides easy blog post management:

1. **Access**: Navigate to `/admin` (e.g., http://localhost:8082/admin)
2. **Login**: Use credentials from `ADMIN_USER` and `ADMIN_PASS` environment variables
3. **Manage Posts**:
   - Click any post in the sidebar to edit
   - Click "New Post" to create new content
   - Fill in title, category, summary, content, and tags
   - Click "Save Post" to write changes to `config/config.yaml`
   - Click "Delete" to remove posts

**Features**:
- Live editor with character counters
- Tag management (press Enter to add tags)
- Category selection dropdown
- Automatic saving to config file
- Session-based authentication (24-hour sessions)

### Configuration File

Direct editing of `config/config.yaml` is also supported:

```yaml
services:
  - name: "Service Name"
    description: "What it does"
    url: "https://service.example.com"
    tech: "Technology stack"
    status: "live" # or "development"
    icon: "🎯"

posts:
  - id: "post-slug"
    title: "Post Title"
    date: 2024-12-01T00:00:00Z
    category: "Category"
    summary: "Brief description"
    content: |
      Full post content in Markdown-like format
    tags: ["tag1", "tag2"]
```

## Project Structure

```
homelabsite/
├── main.go                  # Application entry point
├── go.mod                   # Go dependencies
├── config/
│   └── config.yaml         # Services and blog posts
├── web/
│   ├── templates/          # HTML templates
│   │   ├── base.html
│   │   ├── home.html
│   │   ├── services.html
│   │   ├── blog.html
│   │   ├── post.html
│   │   └── about.html
│   └── static/
│       └── css/
│           └── style.css   # Styling
├── charts/
│   └── app/                # Helm chart
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
├── ci/
│   └── kubernetes/
│       ├── kaniko.yaml     # Container build
│       └── trivy.yaml      # Security scanning
├── Dockerfile              # Multi-stage build
├── Jenkinsfile             # CI/CD pipeline
└── Makefile                # Build automation
```

## Configuration

### Environment Variables

- `PORT`: HTTP server port (default: 8080)
- `ADMIN_USER`: Admin username for content management (default: admin)
- `ADMIN_PASS`: Admin password for content management (default: changeme)

**Security Note**: Always set custom credentials in production environments.

```bash
# Set custom credentials
export ADMIN_USER=your_username
export ADMIN_PASS=your_secure_password
go run .
```

### Helm Configuration

Key configurations in `charts/app/values.yaml`:

```yaml
image:
  repository: tinorodney/homelabsite
  tag: v0.0.1

ingress:
  enabled: true
  hosts:
    - host: atarnet.org

resources:
  limits:
    cpu: 200m
    memory: 128Mi
```

## Testing

```bash
# Run tests
go test -v ./...

# Check health endpoint
curl http://localhost:8080/health
```

## Features

- Responsive design optimized for all devices
- Modern dark theme for developer-friendly reading
- Fast page loads with HTMX progressive enhancement
- Blog with category filtering and tag system
- Service showcase with live status indicators
- Health check endpoint for monitoring
- RESTful JSON API for services and posts
- Embedded static files (single binary deployment)
- Production-ready Kubernetes deployment manifests
- Automated CI/CD pipeline with security scanning
- Admin panel for content management
- Markdown-like content rendering

## Security

- Trivy scanning in CI/CD pipeline catches vulnerabilities before deployment
- Distroless base image minimizes attack surface
- Read-only container filesystem
- Kubernetes resource limits prevent resource exhaustion
- HTTPS/TLS encryption via NGINX Ingress and Let's Encrypt
- No sensitive data embedded in container images
- Session-based authentication for admin panel

## Learn More

Read about the infrastructure and development on the blog:
- [Deep Dive: My Homelab Architecture and Service Interactions](https://atarnet.org/blog/homelab-architecture-deep-dive)
- [Building a Kubernetes Homelab](https://atarnet.org/blog/kubernetes-homelab-journey)
- [Building CI/CD Pipelines with Jenkins on Kubernetes](https://atarnet.org/blog/jenkins-kubernetes-cicd)
- [Why I Choose Go for My Homelab Services](https://atarnet.org/blog/golang-web-services)

---

Built as part of my Kubernetes homelab infrastructure.
