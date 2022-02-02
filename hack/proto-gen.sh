#!/bin/sh

ROOT_DIR=$(cd $(dirname $BASH_SOURCE)/..; pwd)
PATH="$ROOT_DIR/dist:$PATH"
MOD_ROOT=$(go env GOPATH)/pkg/mod
PROTO_FILES=$(find $ROOT_DIR \( -name "*.proto" -and -path '*/server/*' \) | sort)

for i in ${PROTO_FILES}; do
    GOOGLE_PROTO_API_PATH=$MOD_ROOT/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis
    GOGO_PROTOBUF_PATH=$ROOT_DIR/vendor/github.com/gogo/protobuf
    protoc \
        -I$ROOT_DIR \
        -I/usr/local/include \
        -I./vendor \
        -I$GOPATH/src \
        -I$GOOGLE_PROTO_API_PATH \
        -I$GOGO_PROTOBUF_PATH \
        --${GO_PROTO_GEN}_out=plugins=grpc:$GOPATH/src \
        --grpc-gateway_out=logtostderr=true:$GOPATH/src \
        --swagger_out=logtostderr=true:. \
        $i
done

SWAGGER_ROOT=server
COLLISIONS="30"
SWAGGER_DEST="$OPENAPI_DEST"
PRIMARY_SWAGGER=$(mktemp)
COMB_SWAGGER=$(mktemp)

cat <<EOF > "${PRIMARY_SWAGGER}"
{
"swagger": "2.0",
"info": {
"title": "TemplateHub registry API Specification",
"description": "TemplateHub registry API Specification version: ${VERSION}",
"version": "${VERSION}"
},
"paths": {}
}
EOF

rm -f "${SWAGGER_DEST}"
find "${SWAGGER_ROOT}" -name '*.swagger.json' -exec swagger mixin -c "${COLLISIONS}" "${PRIMARY_SWAGGER}" '{}' \+ > "${COMB_SWAGGER}"
jq -r 'del(.definitions[].properties[]? | select(."$ref"!=null and .description!=null).description) | del(.definitions[].properties[]? | select(."$ref"!=null and .title!=null).title)' "${COMB_SWAGGER}" > "${SWAGGER_DEST}"

/bin/rm "${PRIMARY_SWAGGER}" "${COMB_SWAGGER}"

find "${SWAGGER_ROOT}" -name '*.swagger.json' -delete
