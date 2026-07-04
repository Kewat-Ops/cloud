pipeline {
    agent { label 'kubernets-agent' }

    environment {
        DOCKER_CREDS = credentials('dockerhub-creds')
        DOCKER_USERNAME = "${DOCKER_CREDS_USR}"
        GIT_COMMIT = "${env.GIT_COMMIT}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Host-Info') {
            steps {
                sh '''
                k3s --version
                hostname
                whoami
                pwd
                '''
            }
        }

        stage('Building & Pushing') {
            steps {
                sh '''
                echo $DOCKER_CREDS_PSW | docker login -u $DOCKER_CREDS_USR --password-stdin
                '''
                sh '''
                docker build -t $DOCKER_CREDS_USR/cloud-python:latest ./python-dir -f ./python-dir/dockerfile-python
                docker push $DOCKER_CREDS_USR/cloud-python:latest
                '''
                sh '''
                docker build -t $DOCKER_CREDS_USR/cloud-node:latest ./node-dir -f ./node-dir/dockerfile-node
                docker push $DOCKER_CREDS_USR/cloud-node:latest
                '''
                sh '''
                docker build -t $DOCKER_CREDS_USR/cloud-go:latest ./go-dir -f ./go-dir/dockerfile-go
                docker push $DOCKER_CREDS_USR/cloud-go:latest
                '''
            }
        }

        stage('Image Scanning') {
            steps {
                sh 'trivy image $DOCKER_CREDS_USR/cloud-python:latest --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 || true'
                sh 'trivy image $DOCKER_CREDS_USR/cloud-node:latest --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 || true'
                sh 'trivy image $DOCKER_CREDS_USR/cloud-go:latest --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 || true'
            }
        }

        stage('Testing & Promoting Stable Images') {
            steps {
                sh '''
                cd python-dir && python3 -m venv venv && . venv/bin/activate && pip install -r requirements.txt && pytest tests
                '''
                sh '''
                docker tag $DOCKER_CREDS_USR/cloud-python:latest $DOCKER_CREDS_USR/cloud-python:stable
                docker push $DOCKER_CREDS_USR/cloud-python:stable
                '''
                sh '''
                cd ../node-dir && npm install && npm test
                '''
                sh '''
                docker tag $DOCKER_CREDS_USR/cloud-node:latest $DOCKER_CREDS_USR/cloud-node:stable
                docker push $DOCKER_CREDS_USR/cloud-node:stable
                '''
                sh '''
                cd ../go-dir && go test ./...
                '''
                sh '''
                docker tag $DOCKER_CREDS_USR/cloud-go:latest $DOCKER_CREDS_USR/cloud-go:stable
                docker push $DOCKER_CREDS_USR/cloud-go:stable
                '''
            }
        }

        stage('Deployment') {
            steps {
                sh '''
                echo "Applying Kubernetes manifests..."
                kubectl apply -f kubernets.yml
                '''
            }
        }

        stage('Rollback & Healthcheck') {
            steps {
                sh '''
                for SVC in python node go; do
                  echo "Checking $SVC..."
                  status=$(kubectl get pods -l app=${SVC}-app -o jsonpath='{.items[*].status.phase}')
                  if echo "$status" | grep -q "Failed"; then
                    echo "$SVC failed ---> Rolling back to stable"
                    kubectl set image deployment/${SVC}-deployment ${SVC}-container=$DOCKER_CREDS_USR/cloud-${SVC}:stable
                  else
                    echo "$SVC is healthy"
                  fi
                done
                '''
            }
        }
    }
}
