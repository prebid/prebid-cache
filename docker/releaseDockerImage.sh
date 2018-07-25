#!/bin/sh -xe
#You need to be in dokcer directory to execute this script
#cd ${WORKSPACE}
#cd ../


#sudo docker login -u $DOCKER_USERNAME -p $DOCKER_PASSWORD -e $DOCKER_EMAIL  docker.pubmatic.com
#echo $DOCKER_USERNAME : $DOCKER_PASSWORD : $DOCKER_EMAIL
#echo $GROUP_ID : $ARTIFACT_ID : $ARTIFACT_VERSION : $FINAL_ARTIFACT_NAME : $ARTIFACT_PACKAGING_TYPE ;

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
