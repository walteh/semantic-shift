#!/bin/bash
set -e

# Script to generate .mockery.yaml configuration by scanning for //go:mockery comments
# in Go files and identifying interfaces that need to be mocked.

# Start with a clean temporary file for our output
TMP_FILE=$(mktemp)
PACKAGES_FILE=$(mktemp)

# Find all Go files with a "//go:mockery" comment
echo "Searching for files with //go:mockery comments..."
FILES=$(grep -r "//go:mockery" --include="*.go" . | cut -d: -f1 | sort | uniq)

# Get the main module path (first line only)
MODULE_PATH=$(go list -m | head -n 1)
echo "Using module path: $MODULE_PATH"

# Keep track of packages we've already added
ADDED_PACKAGES=()

# Process each file and build packages section
for FILE in $FILES; do
	echo "Processing file: $FILE"

	# Get package name
	PKG_NAME=$(grep -m 1 "^package " "$FILE" | cut -d" " -f2)

	# Calculate relative path (remove leading ./ if present)
	REL_PATH=$(dirname "$FILE" | sed 's|^\./||')

	# Full import path
	IMPORT_PATH="$MODULE_PATH/$REL_PATH"

	# Check if we've already added this package
	if [[ " ${ADDED_PACKAGES[@]} " =~ " ${IMPORT_PATH} " ]]; then
		echo "  Skipping already added package: $IMPORT_PATH"
		continue
	fi

	# Get interface names that follow a "//go:mockery" comment
	echo "  Looking for interfaces in $FILE..."
	INTERFACES=$(grep -A 1 "//go:mockery" "$FILE" | grep -E "type [A-Za-z0-9_]+ interface" | sed -E 's/type ([A-Za-z0-9_]+) interface.*/\1/')

	if [ -n "$INTERFACES" ]; then
		echo "  Found interfaces: $INTERFACES"
		# Add package to mockery config packages
		echo "    $IMPORT_PATH:" >>"$PACKAGES_FILE"
		echo "        interfaces:" >>"$PACKAGES_FILE"

		# Add each interface
		for INTERFACE in $INTERFACES; do
			echo "            $INTERFACE: {}" >>"$PACKAGES_FILE"
		done

		# Mark package as added
		ADDED_PACKAGES+=("$IMPORT_PATH")
	fi
done

# Now extract the header from the existing .mockery.yaml
echo "Updating .mockery.yaml with generated packages..."
if grep -q "packages: #generated" .mockery.yaml; then
	# Extract everything up to the packages line
	grep -B 100 "packages: #generated" .mockery.yaml | head >"$TMP_FILE"
	# Add our marker line
	echo "dir: gen/mockery" >>"$TMP_FILE"
	echo "packages: #generated" >>"$TMP_FILE"
	# Add the generated packages
	cat "$PACKAGES_FILE" >>"$TMP_FILE"
else
	# If the file doesn't have the marker, use a default template
	cat >"$TMP_FILE" <<EOF
inpackage: false
with-expecter: true
testonly: false
exported: true
dir: gen/mockery
outpkg: mockery
resolve-type-alias: false
issue-845-fix: true
filename: "{{.InterfaceName}}.{{.PackageName}}.mockery.go"
mockname: Mock{{.InterfaceName}}_{{.PackageName}}
packages: #generated
EOF
	# Add the generated packages
	cat "$PACKAGES_FILE" >>"$TMP_FILE"
fi

# Display the generated config
echo "Generated mockery config:"
cat "$TMP_FILE"

mkdir -p gen/mockery

# Replace the original mockery.yaml with the new one
mv "$TMP_FILE" gen/mockery/.mockery.yaml

echo "Updated .mockery.yaml with interfaces from the codebase"

# Clean up temporary file
rm -f "$PACKAGES_FILE"
