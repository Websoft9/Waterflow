# Getting Started with Waterflow

Welcome to Waterflow! This guide will walk you through creating your first workflow and understanding the basic concepts.

## 🎯 What You'll Learn

By the end of this guide, you'll be able to:
- Create a simple Waterflow workflow
- Run it locally
- Understand basic workflow concepts
- Deploy to different environments

## 🚀 Quick Start

### 1. Installation Check

First, ensure Waterflow is installed:

```bash
waterflow version
```

If not installed, see the [Installation Guide](installation.md).

### 2. Initialize Workspace

Create a new directory for your workflows:

```bash
mkdir my-first-workflow
cd my-first-workflow

# Initialize Waterflow workspace
waterflow init
```

This creates:
- `workflows/` directory for workflow files
- `config/` directory for configuration
- Example workflow files

## 📝 Your First Workflow

Let's create a simple "Hello World" workflow that demonstrates basic concepts.

### Create the Workflow File

Create `workflows/hello-world.yaml`:

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: hello-world
  description: "A simple hello world workflow"

spec:
  stages:
    - name: greet
      jobs:
        - name: say-hello
          container: alpine:latest
          commands:
            - echo "Hello, Waterflow!"
            - echo "Current date: $(date)"
            - echo "Workflow executed successfully!"
```

### Understanding the Structure

- **`apiVersion`**: Specifies the Waterflow API version
- **`kind`**: Type of resource (Workflow)
- **`metadata`**: Information about the workflow (name, description)
- **`spec`**: The actual workflow specification
- **`stages`**: Sequential phases of execution
- **`jobs`**: Individual tasks within a stage

## ▶️ Running Your First Workflow

### Validate the Workflow

Before running, validate the YAML syntax:

```bash
waterflow validate workflows/hello-world.yaml
```

You should see:
```
✅ Workflow validation successful
```

### Execute Locally

Run the workflow:

```bash
waterflow run workflows/hello-world.yaml
```

Expected output:
```
🚀 Starting workflow: hello-world
📦 Executing stage: greet
🏃 Running job: say-hello
Hello, Waterflow!
Current date: Wed Dec 5 10:30:00 UTC 2025
Workflow executed successfully!
✅ Workflow completed successfully
```

### Run with Verbose Output

For more detailed execution information:

```bash
waterflow run --verbose workflows/hello-world.yaml
```

## 🔄 Adding Dependencies

Let's create a more complex workflow with dependencies.

Create `workflows/multi-stage.yaml`:

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: multi-stage-example
  description: "Multi-stage workflow with dependencies"

spec:
  stages:
    - name: prepare
      jobs:
        - name: setup-environment
          container: alpine:latest
          commands:
            - echo "Setting up environment..."
            - mkdir -p /tmp/workflow-data
            - echo "Environment ready" > /tmp/workflow-data/status.txt

    - name: build
      dependsOn: [prepare]  # Wait for prepare stage to complete
      jobs:
        - name: compile-app
          container: golang:1.21-alpine
          commands:
            - echo "Building application..."
            - echo "Build completed" >> /tmp/workflow-data/status.txt

    - name: test
      dependsOn: [build]  # Wait for build stage to complete
      jobs:
        - name: run-tests
          container: golang:1.21-alpine
          commands:
            - echo "Running tests..."
            - echo "All tests passed!" >> /tmp/workflow-data/status.txt

    - name: deploy
      dependsOn: [test]  # Wait for test stage to complete
      jobs:
        - name: deploy-app
          container: alpine:latest
          commands:
            - echo "Deploying application..."
            - cat /tmp/workflow-data/status.txt
            - echo "Deployment completed successfully!"
```

### Run the Multi-Stage Workflow

```bash
waterflow run workflows/multi-stage.yaml
```

Notice how stages execute in order: `prepare` → `build` → `test` → `deploy`.

## 📊 Parallel Execution

Waterflow can run jobs within a stage in parallel. Create `workflows/parallel.yaml`:

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: parallel-execution
  description: "Demonstrating parallel job execution"

spec:
  stages:
    - name: parallel-stage
      jobs:
        - name: job-a
          container: alpine:latest
          commands:
            - echo "Starting job A"
            - sleep 2
            - echo "Job A completed"

        - name: job-b
          container: alpine:latest
          commands:
            - echo "Starting job B"
            - sleep 1
            - echo "Job B completed"

        - name: job-c
          container: alpine:latest
          commands:
            - echo "Starting job C"
            - sleep 3
            - echo "Job C completed"
```

Run it:

```bash
waterflow run workflows/parallel.yaml
```

Notice that jobs B completes before A and C, demonstrating parallel execution.

## 🔧 Using Environment Variables

Create `workflows/env-vars.yaml`:

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: environment-variables
  description: "Using environment variables in workflows"

spec:
  env:
    - name: APP_NAME
      value: "MyWaterflowApp"
    - name: VERSION
      value: "1.0.0"
    - name: ENVIRONMENT
      value: "development"

  stages:
    - name: demo
      jobs:
        - name: show-env
          container: alpine:latest
          commands:
            - echo "Application: $APP_NAME"
            - echo "Version: $VERSION"
            - echo "Environment: $ENVIRONMENT"
            - echo "Current user: $(whoami)"
            - echo "Working directory: $(pwd)"
```

Run it:

```bash
waterflow run workflows/env-vars.yaml
```

## 📁 Working with Files

Create `workflows/file-operations.yaml`:

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: file-operations
  description: "Demonstrating file operations and artifacts"

spec:
  stages:
    - name: create-files
      jobs:
        - name: generate-config
          container: alpine:latest
          commands:
            - mkdir -p output
            - echo "server:" > output/config.yaml
            - echo "  host: localhost" >> output/config.yaml
            - echo "  port: 8080" >> output/config.yaml
            - echo "Config file created"

    - name: process-files
      dependsOn: [create-files]
      jobs:
        - name: validate-config
          container: alpine:latest
          commands:
            - echo "Validating configuration..."
            - cat output/config.yaml
            - echo "Configuration is valid"

        - name: create-archive
          container: alpine:latest
          commands:
            - tar -czf output/config.tar.gz output/config.yaml
            - echo "Archive created: $(ls -la output/)"
```

Run it:

```bash
waterflow run workflows/file-operations.yaml
```

## 🚀 Deploying Workflows

### Local Deployment

```bash
# Run in background
waterflow run --detach workflows/hello-world.yaml

# Check status
waterflow status

# View logs
waterflow logs

# Stop workflow
waterflow stop
```

### Docker Deployment

```bash
# Build custom image with your workflow
waterflow build --image my-workflow:latest workflows/hello-world.yaml

# Run in Docker
docker run --rm my-workflow:latest
```

### Kubernetes Deployment (Advanced)

```bash
# Deploy to Kubernetes cluster
waterflow deploy --platform kubernetes workflows/hello-world.yaml

# Check deployment status
kubectl get pods
kubectl logs -l app=waterflow-workflow
```

## 🔍 Monitoring and Debugging

### View Workflow History

```bash
# List all workflows
waterflow list

# Show workflow details
waterflow show hello-world

# View execution logs
waterflow logs hello-world
```

### Debug Mode

```bash
# Run with debug output
waterflow run --debug workflows/hello-world.yaml

# Dry run (validate without executing)
waterflow run --dry-run workflows/hello-world.yaml
```

## 🎉 Next Steps

Now that you understand the basics, explore:

- **[Workflow Configuration](workflow-config.md)**: Advanced YAML features
- **[CLI Reference](cli-reference.md)**: All command-line options
- **[Deployment Strategies](deployment-strategies.md)**: Production deployment patterns
- **[Plugin Development](plugin-development.md)**: Extending Waterflow

## 🆘 Getting Help

- **Documentation**: [docs.websoft9.com/waterflow](https://docs.websoft9.com/waterflow)
- **Examples**: Check `examples/` directory for more workflows
- **Community**: [GitHub Discussions](https://github.com/Websoft9/Waterflow/discussions)
- **Issues**: [GitHub Issues](https://github.com/Websoft9/Waterflow/issues)

---

*Happy flowing with Waterflow! 🌊*

*Last updated: December 5, 2025*