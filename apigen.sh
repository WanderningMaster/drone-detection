cd api
bash protogen.sh
cp -r ./gen/go/* ../audio-server
cp -r ./gen/go/* ../gateway
cp -r ./gen/python/* ../analyzer
