pipeline {
  agent any

  environment {
    GITHUB_USER     = "tinotenda-alfaneti"
    REPO_NAME       = "${env.JOB_NAME.split('/')[1]}"
    IMAGE_NAME      = "tinorodney/${REPO_NAME}"
    TAG             = "v0.0.6"
    APP_NAME        = "${REPO_NAME}"
    NAMESPACE       = "${REPO_NAME}-ns"
    SOURCE_NS       = "test-ns"
    KUBECONFIG_CRED = "kubeconfigglobal"
    PATH            = "$WORKSPACE/go/bin:$WORKSPACE/bin:$PATH"
    GOPATH          = "$WORKSPACE/gopath"
  }

  stages {

    stage('Checkout Code') {
      steps {
        echo "Checking out ${REPO_NAME}..."
        checkout scm
        sh 'mkdir -p $WORKSPACE/bin'
      }
    }

    stage('Install Tools') {
      steps {
        sh '''
          echo "Installing kubectl & helm..."
          ARCH=$(uname -m)
          case "$ARCH" in
              x86_64)   KARCH=amd64 ;;
              aarch64)  KARCH=arm64 ;;
              armv7l)   KARCH=armv7 ;;
              *) echo "Unsupported arch: $ARCH" && exit 1 ;;
          esac

          # Kubectl
          VER=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)
          curl -sLO https://storage.googleapis.com/kubernetes-release/release/${VER}/bin/linux/${KARCH}/kubectl
          chmod +x kubectl && mv kubectl $WORKSPACE/bin/

          # Helm
          HELM_VER="v3.14.4"
          curl -sLO https://get.helm.sh/helm-${HELM_VER}-linux-${KARCH}.tar.gz
          tar -zxf helm-${HELM_VER}-linux-${KARCH}.tar.gz
          mv linux-${KARCH}/helm $WORKSPACE/bin/helm
          chmod +x $WORKSPACE/bin/helm
          rm -rf linux-${KARCH} helm-${HELM_VER}-linux-${KARCH}.tar.gz
        '''
      }
    }

    stage('Install Go') {
      steps {
        echo 'Installing Go toolchain...'
        sh '''
          GO_VERSION=1.22.0
          ARCH=$(uname -m)
          case "$ARCH" in
            x86_64)  GO_ARCH=amd64 ;;
            aarch64) GO_ARCH=arm64 ;;
            arm64)   GO_ARCH=arm64 ;;
            armv7l)  GO_ARCH=armv6l ;;
            *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
          esac

          curl -sSLo go.tar.gz https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz
          rm -rf "$WORKSPACE/go"
          tar -C "$WORKSPACE" -xzf go.tar.gz
          rm go.tar.gz

          # Install golangci-lint
          echo "Installing golangci-lint..."
          GOLANGCI_VER="v1.62.2"
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $WORKSPACE/bin ${GOLANGCI_VER}
        '''
      }
    }

    stage('Lint Code') {
      steps {
        sh '''
          echo "Running golangci-lint..."
          export GOROOT=$WORKSPACE/go
          export GOCACHE=$WORKSPACE/.cache/go-build
          export PATH=$GOROOT/bin:$PATH
          
          # Debug: check module info
          echo "Checking go.mod..."
          ls -la go.mod
          cat go.mod | head -5
          
          echo "Checking go environment..."
          $WORKSPACE/go/bin/go env GOMOD
          $WORKSPACE/go/bin/go env GOROOT
          
          # Download dependencies
          $WORKSPACE/go/bin/go mod download
          
          # Verify module is working
          $WORKSPACE/go/bin/go list ./...
          
          # Run linter
          $WORKSPACE/bin/golangci-lint run --out-format colored-line-number
        '''
      }
    }

    stage('Run Tests') {
      steps {
        sh '''
          echo "Running Go tests..."
          export GOROOT=$WORKSPACE/go
          export GOCACHE=$WORKSPACE/.cache/go-build
          export PATH=$GOROOT/bin:$PATH
          
          cd $WORKSPACE
          $WORKSPACE/go/bin/go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
        '''
      }
    }

    stage('Verify Cluster Access') {
      steps {
        withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
          sh '''
            echo "Setting up kubeconfig..."
            mkdir -p $WORKSPACE/.kube
            cp "$KUBECONFIG_FILE" $WORKSPACE/.kube/config
            chmod 600 $WORKSPACE/.kube/config
            export KUBECONFIG=$WORKSPACE/.kube/config
            $WORKSPACE/bin/kubectl cluster-info
          '''
        }
      }
    }

    stage('Prepare Namespace & Secrets') {
      steps {
        withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
          sh '''
            export KUBECONFIG=$WORKSPACE/.kube/config
            echo "Ensuring namespace ${NAMESPACE} exists..."
            $WORKSPACE/bin/kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

            echo "Copying dockerhub-creds from ${SOURCE_NS} to ${NAMESPACE}..."
            $WORKSPACE/bin/kubectl get secret dockerhub-creds -n ${SOURCE_NS} -o yaml \
              | grep -v 'resourceVersion:' | grep -v 'uid:' | grep -v 'creationTimestamp:' \
              | sed "s/namespace: ${SOURCE_NS}/namespace: ${NAMESPACE}/" \
              | $WORKSPACE/bin/kubectl apply -n ${NAMESPACE} -f -

            echo "Ensuring kaniko-builder ServiceAccount and RBAC exist in ${NAMESPACE}..."
            $WORKSPACE/bin/kubectl get sa kaniko-builder -n ${NAMESPACE} >/dev/null 2>&1 \
              || $WORKSPACE/bin/kubectl create serviceaccount kaniko-builder -n ${NAMESPACE}

            CRB_NAME="kaniko-builder-${NAMESPACE}"
            if ! $WORKSPACE/bin/kubectl get clusterrolebinding ${CRB_NAME} >/dev/null 2>&1; then
              $WORKSPACE/bin/kubectl create clusterrolebinding ${CRB_NAME} \
                --clusterrole=cluster-admin \
                --serviceaccount=${NAMESPACE}:kaniko-builder
            fi

            echo "Namespace setup complete for ${NAMESPACE}"
          '''
        }
      }
    }



    stage('Build Image with Kaniko') {
      steps {
        withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
          sh '''
            export KUBECONFIG=$WORKSPACE/.kube/config
            echo "Launching Kaniko build for ${REPO_NAME}..."

            CONTEXT_URL="git://github.com/${GITHUB_USER}/${REPO_NAME}.git"
            IMAGE_DEST="${IMAGE_NAME}:${TAG}"
            echo "Using context: $CONTEXT_URL"
            echo "Destination: $IMAGE_DEST"

            sed -e "s|__CONTEXT_URL__|${CONTEXT_URL}|g" \
                -e "s|__IMAGE_DEST__|${IMAGE_DEST}|g" \
                $WORKSPACE/ci/kubernetes/kaniko.yaml > kaniko-job.yaml

            $WORKSPACE/bin/kubectl delete job kaniko-job -n ${NAMESPACE} --ignore-not-found=true
            $WORKSPACE/bin/kubectl apply -f kaniko-job.yaml -n ${NAMESPACE}
            $WORKSPACE/bin/kubectl wait --for=condition=complete job/kaniko-job -n ${NAMESPACE} --timeout=15m
            echo "Kaniko build completed."
            $WORKSPACE/bin/kubectl logs job/kaniko-job -n ${NAMESPACE} || true
          '''
        }
      }
    }

    stage('Scan Image with Trivy') {
      steps {
        withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
          sh '''
            export KUBECONFIG=$WORKSPACE/.kube/config
            echo "Running Trivy vulnerability scan..."

            IMAGE_DEST="${IMAGE_NAME}:${TAG}"
            sed "s|__IMAGE_DEST__|${IMAGE_DEST}|g" $WORKSPACE/ci/kubernetes/trivy.yaml > trivy-job.yaml

            $WORKSPACE/bin/kubectl delete job trivy-scan -n ${NAMESPACE} --ignore-not-found=true
            $WORKSPACE/bin/kubectl apply -f trivy-job.yaml -n ${NAMESPACE}
            $WORKSPACE/bin/kubectl wait --for=condition=complete job/trivy-scan -n ${NAMESPACE} --timeout=5m || true
            echo "Trivy scan results:"
            $WORKSPACE/bin/kubectl logs job/trivy-scan -n ${NAMESPACE} || true
          '''
        }
      }
    }

    stage('Deploy with Helm') {
      steps {
        withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
          sh '''
            export KUBECONFIG=$WORKSPACE/.kube/config
            echo "Deploying ${APP_NAME} via Helm..."

            # Generate values file with embedded data
            bash $WORKSPACE/scripts/generate-helm-values.sh

            # Deploy with Helm
            $WORKSPACE/bin/helm upgrade --install ${APP_NAME} $WORKSPACE/charts/app \
              --namespace ${NAMESPACE} \
              --create-namespace \
              --set image.repository=${IMAGE_NAME} \
              --set image.tag=${TAG} \
              -f /tmp/helm-data-values.yaml \
              --wait --timeout 5m

            echo "Deployment complete!"

          '''
        }
      }
    }

    stage('Verify Deployment') {
      steps {
        withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
          sh '''
            export KUBECONFIG=$WORKSPACE/.kube/config
            echo "Running Helm test for ${APP_NAME}..."
            $WORKSPACE/bin/helm test ${APP_NAME} --namespace ${NAMESPACE} --logs --timeout 2m
          '''
        }
      }
    }

  }

  post {
    success {
      echo "✅ Pipeline completed successfully."
    }
    failure {
      echo "❌ Pipeline failed."
    }
    always {
      echo "Cleaning up Kubernetes jobs..."
      withCredentials([file(credentialsId: "${KUBECONFIG_CRED}", variable: 'KUBECONFIG_FILE')]) {
        sh '''
          export KUBECONFIG=$WORKSPACE/.kube/config
          kubectl delete job kaniko-job -n ${NAMESPACE} --ignore-not-found=true
          kubectl delete job trivy-scan -n ${NAMESPACE} --ignore-not-found=true
          echo "Cleanup complete."
        '''
      }
      cleanWs()
    }
  }
}
