# Waterflow Concepts

This guide explains the core concepts and terminology used in Waterflow. Understanding these concepts will help you design and manage effective workflows.

## 📋 Core Concepts

### Workflow

A **Workflow** is the top-level construct in Waterflow. It defines a complete automation process that can be executed as a unit.

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: my-workflow
  description: "Description of what this workflow does"
spec:
  # Workflow specification
```

**Key Characteristics:**
- **Atomic**: Workflows execute as complete units
- **Versioned**: Each workflow has a version and can be tracked
- **Reusable**: Workflows can be parameterized and reused
- **Observable**: Execution status and logs are captured

### Stage

A **Stage** represents a logical phase in a workflow execution. Stages execute sequentially and contain one or more jobs.

```yaml
stages:
  - name: build
    jobs:
      - name: compile
        # job definition
  - name: test
    dependsOn: [build]  # Wait for build stage
    jobs:
      - name: unit-tests
        # job definition
```

**Stage Properties:**
- **Sequential**: Stages run one after another
- **Dependency-aware**: Can depend on other stages
- **Parallel jobs**: Jobs within a stage can run in parallel
- **Failure handling**: Stage failure can stop workflow execution

### Job

A **Job** is the smallest unit of work in Waterflow. It defines a specific task to be executed.

```yaml
jobs:
  - name: my-job
    container: golang:1.21
    commands:
      - go build -o app .
      - ./app --version
    env:
      - name: GOOS
        value: linux
```

**Job Characteristics:**
- **Isolated**: Each job runs in its own container
- **Configurable**: Custom environment, resources, and commands
- **Observable**: Individual job status and logs
- **Retryable**: Can be configured to retry on failure

## 🔄 Execution Model

### Workflow Lifecycle

```
Created → Validated → Scheduled → Running → Completed/Failed
    ↓         ↓         ↓         ↓         ↓
   Draft   Syntax OK  Queued   Executing  Terminal State
```

### Stage Execution

1. **Dependency Check**: Ensure all dependent stages completed successfully
2. **Job Scheduling**: Schedule all jobs in the stage for parallel execution
3. **Execution**: Run jobs concurrently
4. **Completion**: Stage completes when all jobs finish
5. **Status Propagation**: Success/failure affects dependent stages

### Job Execution

1. **Container Creation**: Spin up container with specified image
2. **Environment Setup**: Configure environment variables and volumes
3. **Command Execution**: Run commands sequentially
4. **Cleanup**: Clean up resources regardless of outcome
5. **Result Capture**: Capture exit codes, logs, and artifacts

## 📊 Dependencies

### Stage Dependencies

Stages can depend on other stages using the `dependsOn` field:

```yaml
stages:
  - name: build
    jobs: [...]
  - name: test
    dependsOn: [build]  # Must wait for build
    jobs: [...]
  - name: deploy
    dependsOn: [test]   # Must wait for test
    jobs: [...]
```

**Dependency Rules:**
- Dependencies create a DAG (Directed Acyclic Graph)
- Circular dependencies are not allowed
- Failed dependencies prevent dependent stages from running

### Job Dependencies

Jobs within a stage can have implicit dependencies through shared resources, but explicit job-level dependencies are not supported. Use separate stages for sequential job execution.

## 🐳 Containerization

### Container Images

Waterflow uses Docker containers to provide consistent execution environments:

```yaml
jobs:
  - name: build
    container: golang:1.21-alpine  # Official image
  - name: test
    container: node:18-slim       # Custom image
  - name: deploy
    container: alpine:latest      # Minimal image
```

### Container Lifecycle

1. **Pull Image**: Download container image if not cached
2. **Create Container**: Instantiate container with job configuration
3. **Mount Volumes**: Attach persistent storage and artifacts
4. **Execute Commands**: Run job commands in container
5. **Capture Output**: Stream logs and capture exit status
6. **Cleanup**: Remove container and temporary resources

### Resource Management

```yaml
jobs:
  - name: heavy-computation
    container: python:3.9
    resources:
      cpu: "2"      # CPU cores
      memory: "4Gi" # Memory limit
      disk: "10Gi"  # Disk space
```

## 🔧 Configuration

### Environment Variables

Environment variables can be set at multiple levels:

```yaml
# Global workflow environment
env:
  - name: ENVIRONMENT
    value: production

stages:
  - name: deploy
    jobs:
      - name: deploy-app
        env:
          - name: DEPLOY_TARGET  # Job-specific
            value: staging
        commands:
          - echo "Deploying to $ENVIRONMENT/$DEPLOY_TARGET"
```

### Secrets Management

Sensitive data is handled through secret references:

```yaml
secrets:
  - name: db-password
    source: vault://secret/database
  - name: api-key
    source: env://API_KEY

jobs:
  - name: deploy
    env:
      - name: DATABASE_PASSWORD
        valueFrom:
          secretKeyRef:
            name: db-password
            key: password
```

## 📁 Artifacts and Volumes

### Artifact Management

Jobs can produce and consume artifacts:

```yaml
stages:
  - name: build
    jobs:
      - name: compile
        artifacts:
          - name: binary
            path: ./bin/app
            type: file

  - name: test
    jobs:
      - name: integration-test
        artifactsFrom:  # Consume artifacts from other jobs
          - job: compile
            artifacts: [binary]
```

### Volume Mounting

Persistent storage can be mounted:

```yaml
jobs:
  - name: process-data
    volumes:
      - name: data-volume
        mountPath: /data
        source: persistent://my-data-volume
    commands:
      - process-data /data/input.txt
```

## 🔄 Execution Strategies

### Sequential Execution

Default behavior where stages run one after another:

```yaml
stages:
  - name: stage1
  - name: stage2  # Runs after stage1 completes
  - name: stage3  # Runs after stage2 completes
```

### Parallel Execution

Jobs within a stage run concurrently:

```yaml
stages:
  - name: parallel-stage
    jobs:
      - name: job1  # Runs in parallel
      - name: job2  # Runs in parallel
      - name: job3  # Runs in parallel
```

### Conditional Execution

Stages can have conditions for execution:

```yaml
stages:
  - name: deploy-production
    when:  # Only run on main branch
      branch: main
    jobs: [...]
```

## 🚨 Error Handling

### Failure Modes

- **Job Failure**: Individual job exits with non-zero code
- **Stage Failure**: Any job in stage fails
- **Workflow Failure**: Any stage fails (unless configured otherwise)

### Retry Logic

Jobs can be configured to retry on failure:

```yaml
jobs:
  - name: flaky-job
    retry:
      maxAttempts: 3
      delay: 10s
      backoffMultiplier: 2.0
    commands:
      - potentially-flaky-command
```

### Failure Handling

Define behavior when failures occur:

```yaml
stages:
  - name: critical-stage
    onFailure: stop  # Options: stop, continue, retry
    jobs: [...]
```

## 📊 Observability

### Logging

Multiple log levels and formats:

```yaml
logging:
  level: info  # debug, info, warn, error
  format: json # json, text
  outputs:
    - stdout
    - file:/var/log/waterflow.log
```

### Metrics

Built-in metrics collection:

- Workflow execution time
- Job success/failure rates
- Resource utilization
- Queue depth and throughput

### Tracing

Distributed tracing support:

```yaml
tracing:
  enabled: true
  provider: jaeger  # jaeger, zipkin, datadog
  serviceName: waterflow-workflow
```

## 🔌 Extensibility

### Plugins

Extend Waterflow functionality through plugins:

```yaml
plugins:
  - name: custom-deployer
    source: https://github.com/example/custom-deployer
    version: v1.0.0
```

### Custom Runtimes

Support for custom execution environments:

```yaml
jobs:
  - name: custom-runtime
    runtime: custom-plugin
    config:
      customParameter: value
```

## 🌐 Multi-Platform Support

### Target Platforms

Waterflow supports multiple deployment targets:

- **Docker**: Local container execution
- **Kubernetes**: Orchestrated container deployment
- **Cloud Providers**: AWS ECS, GCP Cloud Run, Azure Container Instances
- **Bare Metal**: Direct host execution

### Platform-Specific Configuration

```yaml
platforms:
  kubernetes:
    namespace: production
    resources:
      limits:
        cpu: 1000m
        memory: 1Gi
  aws:
    region: us-east-1
    cluster: my-cluster
```

## 🔒 Security Model

### Authentication

Multiple authentication methods:

- **Token-based**: JWT tokens for API access
- **Certificate-based**: mTLS for secure communication
- **OAuth**: Integration with identity providers

### Authorization

Role-based access control:

```yaml
rbac:
  roles:
    - name: admin
      permissions: ["*"]
    - name: developer
      permissions: ["workflow:read", "workflow:execute"]
```

### Audit Logging

Comprehensive audit trail:

```yaml
audit:
  enabled: true
  logLevel: detailed
  retention: 90d
  outputs:
    - elasticsearch
    - s3://audit-logs/
```

## 📈 Performance Considerations

### Optimization Strategies

- **Parallelization**: Maximize concurrent job execution
- **Caching**: Cache container images and dependencies
- **Resource Limits**: Set appropriate CPU/memory limits
- **Batch Processing**: Group similar jobs together

### Scaling Considerations

- **Horizontal Scaling**: Multiple Waterflow instances
- **Queue Management**: Handle large numbers of workflows
- **Storage Optimization**: Efficient artifact storage
- **Network Efficiency**: Minimize data transfer

---

Understanding these core concepts will help you design efficient, maintainable, and scalable workflows with Waterflow.

*Last updated: December 5, 2025*