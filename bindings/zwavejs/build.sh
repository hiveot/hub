# testing build steps
npx esbuild src/main.ts --bundle  --platform=node --target=node20 --preserve-symlinks --outfile=build/zwavejs-esbuild.js  --external:./node_modules/zwave-js/package.json --external:./node_modules/@zwave-js/config/package.json --external:./prebuilds/linux-x64/node.napi.glibc.node
npx postject dist/nzwavejs NODE_SEA_BLOB sea-prep.blob --overwrite --sentinel-fuse NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2

#ZWAVEJS_EXTERNAL_CONFIG=dist/cache node --preserve-symlinks build/zwavejs-esbuild.js --clientID testsvc --home ~/bin/hiveot

dist/nzwavejs --clientID testsvc --home ~/bin/hiveot
