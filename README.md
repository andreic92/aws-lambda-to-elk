# aws-lambda-to-elk

## Description
Go project that is responsible with processing Cloud Watch logs and send them to an ES instance.

## How does it work

### Build it
    GOOS=linux go build -o forwarder

### Run it
    ES_PORT=9200 ES_INDEX=logs ES_HOST=<your_host> ./forwarder 

### Overview
It starts the handler as a lambda function.
During initialization, a check to see if the ES index exists is performed and if it the index does not exist, it tries to create it.
The handler is designed to parse logs received from Cloud Watch, meaning that these has to be decoded (as they are base64 encoded) and unzip the data.
