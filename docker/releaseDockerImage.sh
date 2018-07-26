#!/bin/sh -xe
#You need to be in dokcer directory to execute this script
#cd ${WORKSPACE}
#cd ../

mkdir -p cache
mkdir -p cache/app-resources
cp -rf ../prebid-cache ./cache/
cp -rf ../config-yaml ./cache/
cp -rf ../../../../applications-k8s-config.git/prebid-cache/docker/* ./cache/app-resources/

#Copy the template docker file to actual docker file which will be used for current build.
rm -rf Dockerfile
cp Dockerfile.template Dockerfile

#Replace the place holders with actual values of the current build.
sed "s/GROUP_ID_VAR/$GROUP_ID/g;s/ARTIFACT_ID_VAR/$ARTIFACT_ID/g;s/ARTIFACT_VERSION_VAR/$ARTIFACT_VERSION/g;s/FINAL_ARTIFACT_NAME_VAR/$FINAL_ARTIFACT_NAME/g;s/ARTIFACT_PACKAGING_TYPE_VAR/$ARTIFACT_PACKAGING_TYPE/g;s/APP_RESOURCES_ARTIFACT_VAR/$APP_RESOURCES_ARTIFACT_ID/g"  -i Dockerfile ;

#cat Dockerfile
#Build a docker image.
#sudo docker build  --no-cache=true --tag=docker.pubmatic.com/$ARTIFACT_ID:$ARTIFACT_VERSION .
sudo docker build  --tag=docker.pubmatic.com/$ARTIFACT_ID:$ARTIFACT_VERSION .

# Pushing the image to docker repo
sudo docker push docker.pubmatic.com/$ARTIFACT_ID:$ARTIFACT_VERSION

#Tag this image as a latest image
sudo docker tag -f docker.pubmatic.com/$ARTIFACT_ID:$ARTIFACT_VERSION docker.pubmatic.com/$ARTIFACT_ID:latest
sudo docker push docker.pubmatic.com/$ARTIFACT_ID:latest
