#!/bin/bash

echo "Preparing for deploying..."


sh /propertyReplace.sh ./cache/app-resources/app-resources-$APP_DC-$APP_ENV.yml ./cache/
mv ./cache/app-resources.yml config.yaml
./cache/prebid-cache