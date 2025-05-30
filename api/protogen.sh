PROTO_DIR="./proto"

# Output directories for generated code
OUT_DIR_GO="./gen/go"
OUT_DIR_PY="./gen/python"

rm -rf ./gen

# Ensure output directories exist
mkdir -p "$OUT_DIR_GO"
mkdir -p "$OUT_DIR_PY"

cowsay GENERATE GO FILES
docker run -v $PWD:/defs namely/protoc-all -d "$PROTO_DIR"  -l go -o "$OUT_DIR_GO"


cowsay GENERATE PYTHON FILES
docker run -v $PWD:/defs namely/protoc-all -d "$PROTO_DIR"  -l python -o "$OUT_DIR_PY"

sudo chmod -R 777 ${OUT_DIR_GO}
sudo chmod -R 777 ${OUT_DIR_PY}

rm -f ./gen/__init__.py
