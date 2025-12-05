#!/bin/bash

# Waterflow Build Script
# Builds Waterflow for multiple platforms

set -e

VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
DATE=${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}

echo "🔨 Building Waterflow v$VERSION ($COMMIT)..."

# Clean previous builds
rm -rf bin/
mkdir -p bin/

# Build for multiple platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo $PLATFORM | cut -d'/' -f1)
    ARCH=$(echo $PLATFORM | cut -d'/' -f2)

    BINARY_NAME="waterflow"
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="waterflow.exe"
    fi

    OUTPUT_NAME="bin/waterflow-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME="bin/waterflow-${OS}-${ARCH}.exe"
    fi

    echo "Building for $OS/$ARCH..."

    CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build \
        -ldflags="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.date=$DATE' -s -w" \
        -o $OUTPUT_NAME \
        ./cmd/waterflow

    # Create compressed archive
    if [ "$OS" = "windows" ]; then
        zip -q "${OUTPUT_NAME}.zip" "$OUTPUT_NAME"
        rm "$OUTPUT_NAME"
    else
        tar -czf "${OUTPUT_NAME}.tar.gz" -C bin/ "$(basename $OUTPUT_NAME)"
        rm "$OUTPUT_NAME"
    fi
done

# Create checksums
echo "📋 Generating checksums..."
cd bin/
for file in *; do
    if [[ "$file" == *.tar.gz ]] || [[ "$file" == *.zip ]]; then
        sha256sum "$file" >> checksums.txt
    fi
done
cd ..

echo ""
echo "✅ Build complete!"
echo ""
echo "Artifacts created in bin/:"
ls -la bin/
echo ""
echo "📦 Ready for distribution"