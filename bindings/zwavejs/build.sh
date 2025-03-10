# build for production using esbuild
# the challenge here is how to include the paths to the external resources.
# Nothing seems to work other than using the zwave-js esbuild script.

npm i

#npx esbuild src/main.ts --bundle  --platform=node --target=node20 --preserve-symlinks --sourcemap \
#  --outfile=build/zwavejs-esbuild.js \
#  --external:@zwave-js/config/package.json   \
#  --external:zwave-js/package.json   \
#  --external:./package.json   \
#  --external:@serialport/bindings-cpp/prebuilds    \
#  --external:@zwave-js/config/config \
#  --outfile=build/main.js

#npx postject dist/nzwavejs NODE_SEA_BLOB sea-prep.blob --overwrite --sentinel-fuse NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2

rm -rf dist/* build/*

# use the esbuild from zwave-js-ui, which handles the assets paths,
# and creates a patched package.json to run with pkg inside the build folder.
node ./esbuild.js

cd build
npx pkg package.json -t node20-linux-x64 --out-path ../dist

