#!/usr/bin/env bash
set -e
# build for production using esbuild
# the challenge here is how to include the paths to the external resources.
# Nothing seems to work other than using the zwave-js esbuild script (github.com/zwave-js/zwave-js-ui)

#--- Step 1: Setup
PKG_FOLDER="dist"
echo "Destination folder: ./$PKG_FOLDER"
VERSION=$(node -p "require('./package.json').version")
echo "Version: $VERSION"

NODE_MAJOR=$(node -v | grep -E -o '[0-9].' | head -n 1)

echo "## Clear $PKG_FOLDER folder"
rm -rf $PKG_FOLDER/*

# if --arch is passed as argument, use it as value for ARCH
if [[ "$@" == *"--arch"* ]]; then
	ARCH=$(echo "$@" | grep -oP '(?<=--arch=)[^ ]+')
else
	ARCH=$(uname -m)
fi
echo "## Architecture: $ARCH"


#--- Step 2: build
# use the esbuild from zwave-js-ui, which handles the assets paths,
# and creates a patched package.json to run with pkg inside the build folder.
echo "## Building application..."

#node ./esbuild.cjs  has been replaced
# this just runs tsc to compile typescript
npm run build

#--- Step 3: bundle
echo "## Bundling node$NODE_MAJOR-linux for arch: $ARCH"
# this runs node esbuild.cjs
node esbuild.cjs

cd build
if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
		npx pkg package.json -t node$NODE_MAJOR-linux-arm64 --out-path ../$PKG_FOLDER
elif [ "$ARCH" = "armv7" ]; then
		npx pkg package.json -t node$NODE_MAJOR-linux-armv7 --out-path ../$PKG_FOLDER --public-packages=*
else
		#npx pkg package.json -t node$NODE_MAJOR-linux-x64,node$NODE_MAJOR-win-x64  --out-path ../$PKG_FOLDER
		npx pkg package.json -t node$NODE_MAJOR-linux-x64  --out-path ../$PKG_FOLDER
fi

#npx pkg package.json -t node$NODE_MAJOR-linux-x64 --out-path ../dist
