#!/bin/bash

if [ $# != 2  ] ; then
    echo -e "\n\tHelp : # sh propertyReplace.sh <Absolute filepath of file with values of placeholders> <Absolute path of directory where property placeholder need to replaced> \n"
    exit 1;
fi

config_file=$1
property_directory=$2


while read i
do
    key=`echo $i |  egrep -o ".*="| awk -F= '{print $1}' | sed -r 's/[@.\$\\\/]/\\\&/g'`
    value=`echo $i |  egrep -o "=.*"| sed -r 's/^=//g;s/[@.\$\\\/\&]/\\\&/g'`

    ls $property_directory/* | xargs sed "s/\${$key}/$value/g" -i
done < $config_file