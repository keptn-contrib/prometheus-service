#!/bin/bash

MANIFEST_FILE=./docker/MANIFEST
REPO_URL=https://github.com/$GITHUB_REPOSITORY
JOB_URL=${REPO_URL}/actions/runs/${GITHUB_RUN_ID}

sed -i 's~MANIFEST_REPOSITORY~'"$REPO_URL"'~' ${MANIFEST_FILE}
sed -i 's~MANIFEST_BRANCH~'"${GITHUB_REF##*/}"'~' ${MANIFEST_FILE}
sed -i 's~MANIFEST_COMMIT~'"$GIT_SHA"'~' ${MANIFEST_FILE}
sed -i 's~MANIFEST_TRAVIS_JOB_URL~'"$JOB_URL"'~' ${MANIFEST_FILE}
sed -i 's~MANIFEST_DATE~'"$DATE"'~' ${MANIFEST_FILE}
sed -i 's~MANIFEST_VERSION~'"$VERSION"'~' ${MANIFEST_FILE}
