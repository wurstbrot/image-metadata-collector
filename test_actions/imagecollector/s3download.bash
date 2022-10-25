#!/bin/bash
outputFile="/${TMP_FOLDER}/$1"
amzFile="$1"
bucket="local"
folder="lord-of-the-rings/imagecollector"
resource="/${bucket}/${folder}/${amzFile}"
contentType="application/x-compressed-tar"
dateValue=`date -R`
stringToSign="GET\n\n${contentType}\n${dateValue}\n${resource}"
s3Key="testtesttest"
s3Secret="testtesttest"
signature=`echo -en ${stringToSign} | openssl sha1 -hmac ${s3Secret} -binary | base64`

curl -Ls -H "Host: localhost:9000" \
     -H "Date: ${dateValue}" \
     -H "Content-Type: ${contentType}" \
     -H "Authorization: AWS ${s3Key}:${signature}" \
     http://localhost:9000${resource} -o $outputFile
if [ $? -ne 0 ]; then
  echo "Could not download ${resource}"
fi
