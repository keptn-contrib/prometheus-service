#!/bin/bash
# shellcheck disable=SC2181

VERSION=$1
APP_VERSION=$2
IMAGE=$3

if [ $# -ne 3 ]; then
  echo "Usage: $0 VERSION APP_VERSION IMAGE"
  exit
fi

if [ -z "$VERSION" ]; then
  echo "No Version set, exiting..."
  exit 1
fi

if [ -z "$APP_VERSION" ]; then
  echo "No Image Tag set, defaulting to version"
  APP_VERSION=$VERSION
fi


# replace "appVersion: latest" with "appVersion: $VERSION" in all Chart.yaml files
# find . -name Chart.yaml -exec sed -i -- "s/appVersion: latest/appVersion: ${APP_VERSION}/g" {} \;
# find . -name Chart.yaml -exec sed -i -- "s/version: latest/version: ${VERSION}/g" {} \;

mkdir installer/

# ####################
# HELM CHART
# ####################
BASE_PATH=.
CHARTS_PATH=charts/prometheus-service

helm package ${BASE_PATH}/${CHARTS_PATH} --app-version "$APP_VERSION" --version "$VERSION"
if [ $? -ne 0 ]; then
  echo "Error packaging installer, exiting..."
  exit 1
fi

mv "${IMAGE}-${VERSION}.tgz" "installer/${IMAGE}-${VERSION}.tgz"

#verify the chart
helm template --debug "installer/${IMAGE}-${VERSION}.tgz"

if [ $? -ne 0 ]; then
  echo "::error Helm Chart for ${IMAGE} has templating errors -exiting"
  exit 1
fi

echo "Generated files:"
echo " - installer/${IMAGE}-${VERSION}.tgz"