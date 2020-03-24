#!/bin/bash

CONFIG_FILE_PATH="/app/config.yaml"

sed -i "s%<<REDIS_HOSTNAME>>%${REDIS_HOSTNAME}%g" "${CONFIG_FILE_PATH}"

sed -i "s%<<REDIS_PORT>>%${REDIS_PORT}%g" "${CONFIG_FILE_PATH}"

sed -i "s%<<REDIS_PASSWORD>>%${REDIS_PASSWORD}%g" "${CONFIG_FILE_PATH}"

/app/prebid-cache
