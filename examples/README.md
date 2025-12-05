# Waterflow Examples

This directory contains example workflows demonstrating various Waterflow features and use cases.

## 📁 Examples Overview

### Basic Examples

- **`hello-world.yaml`** - Simple workflow demonstrating basic concepts
  - Single stage with one job
  - Basic container execution
  - Environment variable usage

### Advanced Examples

- **`microservices-deploy.yaml`** - Complete CI/CD pipeline
  - Multi-stage workflow with dependencies
  - Parallel job execution
  - Docker image building and pushing
  - Kubernetes deployment
  - Security scanning integration

## 🚀 Running Examples

### Prerequisites

```bash
# Install Waterflow
curl -sSL https://get.waterflow.io/install.sh | sh

# Or build from source
make build
```

### Run Hello World Example

```bash
# Validate the workflow
waterflow validate examples/hello-world.yaml

# Run the workflow
waterflow run examples/hello-world.yaml
```

### Run Microservices Deployment

```bash
# This example requires Docker and Kubernetes
waterflow validate examples/microservices-deploy.yaml

# Run with custom environment
ENVIRONMENT=production waterflow run examples/microservices-deploy.yaml
```

## 📖 Example Categories

### By Complexity

- **Beginner**: `hello-world.yaml`
- **Intermediate**: Multi-stage workflows
- **Advanced**: Full CI/CD pipelines

### By Use Case

- **CI/CD**: Build, test, deploy pipelines
- **Infrastructure**: Terraform, Ansible integration
- **Microservices**: Multi-service orchestration
- **Data Processing**: ETL and analytics workflows
- **Monitoring**: Observability and alerting

### By Platform

- **Docker**: Container-based workflows
- **Kubernetes**: Orchestrated deployments
- **AWS**: Cloud-native workflows
- **Multi-cloud**: Cross-platform deployments

## 🛠️ Contributing Examples

### Guidelines

1. **Clear Purpose**: Each example should demonstrate specific features
2. **Well Documented**: Include comments explaining complex parts
3. **Realistic**: Use practical scenarios, not toy examples
4. **Tested**: Ensure examples work with current Waterflow version
5. **Minimal**: Keep examples focused and avoid unnecessary complexity

### Adding New Examples

1. Create your example YAML file in this directory
2. Add documentation in the example file comments
3. Update this README with your example
4. Test the example works correctly
5. Submit a pull request

### Example Structure

```yaml
apiVersion: waterflow.io/v1
kind: Workflow
metadata:
  name: example-name
  description: "Brief description of what this example demonstrates"
spec:
  # Workflow specification
  stages:
    - name: stage-name
      jobs:
        - name: job-name
          container: image:tag
          commands:
            - command1
            - command2
```

## 🔍 Understanding Examples

### Key Concepts Demonstrated

- **Workflow Structure**: YAML schema and organization
- **Stage Dependencies**: Sequential execution with `dependsOn`
- **Job Parallelization**: Concurrent execution within stages
- **Environment Variables**: Configuration and parameterization
- **Container Usage**: Image selection and resource management
- **Error Handling**: Retry logic and failure management

### Best Practices

- Use descriptive names for workflows, stages, and jobs
- Include comments for complex configurations
- Leverage environment variables for flexibility
- Implement proper error handling and retries
- Use appropriate container images for tasks

## 📚 Related Documentation

- **[Getting Started](../docs/getting-started.md)** - Basic workflow concepts
- **[Workflow Configuration](../docs/workflow-config.md)** - Advanced YAML features
- **[CLI Reference](../docs/cli-reference.md)** - Command-line interface
- **[Concepts](../docs/concepts.md)** - Core terminology and architecture

## 🤝 Support

- **Issues**: [GitHub Issues](https://github.com/Websoft9/Waterflow/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Websoft9/Waterflow/discussions)
- **Documentation**: [docs.websoft9.com/waterflow](https://docs.websoft9.com/waterflow)

---

*Examples are tested with Waterflow v0.1.0+*