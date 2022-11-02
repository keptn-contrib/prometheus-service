#!/bin/bash

# Fetch latest release and prerelease
prerelease=$(curl -s https://api.github.com/repos/keptn/keptn/releases | jq -r 'map(select(.prerelease)) | sort_by(.tag_name)[-1].tag_name')
release=$(curl --silent "https://api.github.com/repos/keptn/keptn/releases/latest" | jq -r '.tag_name')

# Write variables as output
echo "LATEST_RELEASE=$release" >> "$GITHUB_OUTPUT"
echo "LATEST_PRERELEASE=$prerelease" >> "$GITHUB_OUTPUT"
