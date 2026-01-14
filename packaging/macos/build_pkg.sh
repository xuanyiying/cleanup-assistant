#!/bin/bash
set -e

APP_NAME="cleanup"
VERSION="1.0.0"
IDENTIFIER="com.xuanyiying.cleanup"
BUILD_DIR="../../build"
OUTPUT_DIR="../../dist"
STAGING_DIR="./staging"

# Ensure we are in the script directory
cd "$(dirname "$0")"

# Clean previous build
rm -rf "$STAGING_DIR"
mkdir -p "$STAGING_DIR/usr/local/bin"
mkdir -p "$OUTPUT_DIR"

# Check if binary exists
if [ ! -f "$BUILD_DIR/$APP_NAME-darwin-universal" ]; then
    echo "Error: Universal binary not found. Please run 'make darwin-universal' first."
    exit 1
fi

# Copy binary to staging
cp "$BUILD_DIR/$APP_NAME-darwin-universal" "$STAGING_DIR/usr/local/bin/$APP_NAME"
chmod +x "$STAGING_DIR/usr/local/bin/$APP_NAME"

# Create component package
pkgbuild --root "$STAGING_DIR" \
    --identifier "$IDENTIFIER" \
    --version "$VERSION" \
    --install-location "/" \
    "$OUTPUT_DIR/$APP_NAME-component.pkg"

# Create distribution XML for UI customization
cat > distribution.xml <<EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
    <title>Cleanup CLI $VERSION</title>
    <welcome file="welcome.html"/>
    <license file="../../LICENSE"/>
    <background file="background.png" alignment="bottomleft" scaling="none"/>
    <options customize="never" require-scripts="false"/>
    <pkg-ref id="$IDENTIFIER"/>
    <choices-outline>
        <line choice="default">
            <pkg-ref id="$IDENTIFIER"/>
        </line>
    </choices-outline>
    <choice id="default" title="Cleanup CLI Core">
        <pkg-ref id="$IDENTIFIER"/>
    </choice>
    <pkg-ref id="$IDENTIFIER" version="$VERSION" onConclusion="none">$APP_NAME-component.pkg</pkg-ref>
</installer-gui-script>
EOF

# Create dummy resources if they don't exist
touch welcome.html
# Create a simple background if not exists (optional)

# Build final product package
productbuild --distribution distribution.xml \
    --resources . \
    --package-path "$OUTPUT_DIR" \
    "$OUTPUT_DIR/$APP_NAME-$VERSION.pkg"

# Clean up
rm "$OUTPUT_DIR/$APP_NAME-component.pkg"
rm distribution.xml
rm -rf "$STAGING_DIR"
rm welcome.html

echo "macOS Installer Package created at $OUTPUT_DIR/$APP_NAME-$VERSION.pkg"
