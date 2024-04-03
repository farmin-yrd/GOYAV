#!/bin/bash

CURRENT_DIR="$(dirname $0)"
PORT=${PORT:-8080}
GOYAV_SERVER="http://localhost:${PORT}"

echo -e "\nGoyAV server: ${GOYAV_SERVER}\n"

echo -e "\nUploading eicar test file...\n"

# Upload the eicar test file
response=$(curl -s -X POST ${GOYAV_SERVER}/documents \
                -H "Content-Type: multipart/form-data" \
                -F "tag=eicar file" \
                -F "file=@${CURRENT_DIR}/eicar.com.txt;type=application/octet-stream")

echo ${response} | jq

echo -e "\nGetting antivirus analysis result...\n"

# Get the ID
ID=$(echo $response | jq --raw-output '.id')

# Get the document's status
response=$(curl -s $GOYAV_SERVER/documents/"${ID}")

echo ${response} | jq

