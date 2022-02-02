#!/bin/sh

COMMIT_SHA=$(git rev-parse --verify --quiet HEAD)
GIT_BRANCH=$(git rev-parse --symbolic-full-name --verify --quiet --abbrev-ref HEAD)
GIT_REPO="rkrmr33/template-hub"
FILE="./hack/release_notes.md"

if [[ -z "$PRERELEASE" ]]; then
    PRERELEASE=false
fi

if [[ -z "${VERSION}" ]]; then
    echo "ERROR: VERSION is not set"
    exit 1
fi

if [[ "$(git branch -r --contains $COMMIT_SHA)" != "" ]]; then
    echo "local branch is up to date with remote branch"
else
    echo "ERROR: local brach is not up to date with remote branch, you must push your local changes!"
    exit 1
fi

echo "uploading files:"
    ls -1a ./dist/*.tar.gz ./dist/*.sha256
echo ""

echo "on release branch: $GIT_BRANCH"
echo "running: gh release create ${VERSION} --repo $GIT_REPO -F $FILE -t ${VERSION} --target $GIT_BRANCH --prerelease=$PRERELEASE ./dist/*.tar.gz ./dist/*.sha256"

if [[ "$DRY_RUN" == "1" ]]; then
    exit 0
fi

gh release create ${VERSION} --repo $GIT_REPO -F $FILE -t ${VERSION} --target $GIT_BRANCH --prerelease=$PRERELEASE ./dist/*.tar.gz ./dist/*.sha256

echo "pushing docker images..."
docker tag template-hub quay.io/roikramer120/template-hub:${VERSION}
docker tag template-hub quay.io/roikramer120/template-hub:latest
docker push quay.io/roikramer120/template-hub:${VERSION}