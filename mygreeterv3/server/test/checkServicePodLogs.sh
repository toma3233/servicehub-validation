#!/bin/bash

TEST1='"msg":"finished call"'
TEST2='"code":"OK"'
TEST3='"starting processor receiver"'
TEST4='"processor receiver received [0-9]+ messages from processor loop"' # Match any number of messages
TEST5='"Error getting"' # First part of error for creating the service bus.
TEST6='"CtxLog"' # Ensure that rpcs are made to the servers and we are in the process of serving requests
TEST7='"code":"200"' #Ensure http requests are being served properly
TIMEOUT=150 # in seconds

# check_pod_logs keeps checking the pods logs for desired strings TEST1 and TEST2 until TIMEOUT (in seconds) is reached.
# Inputs:
## NAMESPACE
check_pod_logs() {
    local NAMESPACE=$1
    # Get the start time
    local START_TIME=$(date +%s)
    POD=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    if [ $? -ne 0 ]
    then
        echo "ERROR: error occurred getting pods in $NAMESPACE."
        exit 1
    fi
    echo "Checking $NAMESPACE logs..."
    for (( ; ; ))
    do
        sleep 5
        LOGS_CONTAIN_DESIRED=0
        LOGS_DO_NOT_CONTAIN_TEST5=0
        if [[ $POD == *"async"* ]]; then
            kubectl logs $POD -n $NAMESPACE | grep -E "$TEST3|$TEST4" &> /dev/null
        else
            kubectl logs $POD -n $NAMESPACE | grep "$TEST1" | grep "$TEST2" &> /dev/null
        fi
        if [ $? -eq 0 ]
        then
            LOGS_CONTAIN_DESIRED=1
        fi

        # Check that the service bus error is not in the logs
        if [[ $POD == *"async"* ]]; then
            LOGS_DO_NOT_CONTAIN_TEST5=1
        else
            kubectl logs $POD -n $NAMESPACE | grep "$TEST5" &> /dev/null
        fi
        if [ $? -ne 0 ]
        then
            LOGS_DO_NOT_CONTAIN_TEST5=1
        fi

        # Check all conditions are met.
        if [ $LOGS_CONTAIN_DESIRED -eq 1 ] && [ $LOGS_DO_NOT_CONTAIN_TEST5 -eq 1 ]; then
            echo "$POD has desired logs and does not contain TEST5."
            break
        fi
        local CURRENT_TIME=$(date +%s)
        local TIME_DIFF=$((CURRENT_TIME - START_TIME))
        if ((TIME_DIFF >= $TIMEOUT)); then
            echo "ERROR: The timeout has been reached. $POD does not have desired logs."
            exit 1
        fi
    done
}

# Call the function with the argument "NAMESPACE"
check_pod_logs "servicehubval-mygreeterv3-server"
check_pod_logs "servicehubval-mygreeterv3-demoserver"
check_pod_logs "servicehubval-mygreeterv3-async"
check_pod_logs "servicehubval-mygreeterv3-client"
