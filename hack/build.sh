#!/bin/sh
set -e

if [[ ! -z "${GO_FLAGS}" ]]; then
    echo Building \"${OUT_FILE}\" with flags: \"${GO_FLAGS}\" starting at: \"${MAIN}\"
    for d in ${GO_FLAGS}; do
        export $d
    done
fi

go build -gcflags="${GCFLAGS}" -ldflags=" \
    -X 'github.com/rkrmr33/template-hub/common.Bin=${BINARY_NAME}' \
    -X 'github.com/rkrmr33/template-hub/common.Version=${VERSION}' \
    -X 'github.com/rkrmr33/template-hub/common.GitCommit=${GIT_COMMIT}' \
    -X 'github.com/rkrmr33/template-hub/common.BuildDate=${BUILD_DATE}'" \
    -v -o ${OUT_FILE} ${MAIN}
