#!/bin/bash
# Generate xlarge benchmark YAML file (1000+ lines)
# Used for performance testing: YAML parsing should be < 100ms for 1000 lines

set -e

OUTPUT_FILE="testdata/benchmark/xlarge.yaml"

cat > "$OUTPUT_FILE" << 'EOF'
name: XLarge Benchmark Workflow
on:
  workflow_dispatch:
  push:
    branches: [main, develop]
  pull_request:

env:
  PROJECT_NAME: waterflow
  BUILD_ENV: production
  GO_VERSION: "1.21"
  NODE_VERSION: "20"
  PYTHON_VERSION: "3.11"
  DOCKER_REGISTRY: docker.io
  SERVER_COUNT: "20"

vars:
  timeout_seconds: 300
  retry_attempts: 3
  parallel_limit: 10

jobs:
EOF

# Generate 20 jobs, each with 10 steps (total ~800 lines)
for job in $(seq 0 19); do
  cat >> "$OUTPUT_FILE" << EOF
  job-${job}:
    runs-on: linux-amd64
    timeout-minutes: 30
    needs: $( [ $job -gt 0 ] && echo "[job-$((job-1))]" || echo "[]" )
    env:
      JOB_ID: "job-${job}"
      JOB_INDEX: "${job}"
    steps:
EOF
  
  # Each job has 10 steps
  for step in $(seq 0 9); do
    cat >> "$OUTPUT_FILE" << EOF
      - name: Step ${step} (Job ${job})
        uses: run@v1
        with:
          command: echo "Running job ${job} step ${step}"
EOF
  done
  
  echo "" >> "$OUTPUT_FILE"
done

echo "Generated $OUTPUT_FILE"
wc -l "$OUTPUT_FILE"

# Validate the generated YAML is valid
if command -v yamllint &> /dev/null; then
    echo "Validating YAML syntax..."
    yamllint -d relaxed "$OUTPUT_FILE" || true
else
    echo "yamllint not found, skipping validation"
fi

echo "✅ Done"
