#!/bin/bash

echo "Preparing for deploying..."

ls -l /cache/app-resources
ls -l /cache

echo $APP_ENV
echo $APP_DC
if [ $APP_ENV != "Nothing" ] && [ $APP_DC != "Nothing" ]
    then
        echo "Using APP_ENV and APP_DC environment variables."
        cp /cache/app-resources/prebid-cache_$APP_DC-$APP_ENV.yaml /config.yaml
        ./cache/prebid-cache
        tail -f /var/log/prebidcache/prebid-cache.INFO
    else
        echo "APP_ENV or APP DC environment variables not passed."
fi