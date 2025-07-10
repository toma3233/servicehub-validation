#!/bin/bash

TEST1='"msg":"finished call"'
TEST2='"code":"OK"'
TEST3='"Connected!"'
TIMEOUT=300 # in seconds

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
        kubectl logs $POD -n $NAMESPACE | grep "$TEST1" | grep "$TEST2" > /dev/null 2>&1
        if [ $? -eq 0 ]
        then
            TEST1_TEST2_FOUND=true
        fi

        kubectl logs $POD -n $NAMESPACE | grep "$TEST3" > /dev/null 2>&1
        if [ $? -eq 0 ]
        then
            TEST3_FOUND=true
        fi

        if [ "$TEST1_TEST2_FOUND" = true ] && [ "$TEST3_FOUND" = true ]
        then
            echo "$POD has desired logs for TEST1, TEST2, and TEST3."
            break
        fi

        local CURRENT_TIME=$(date +%s)
        local TIME_DIFF=$((CURRENT_TIME - START_TIME))
        if ((TIME_DIFF >= $TIMEOUT)); then
            echo "ERROR: The timeout has been reached. $POD does not have desired logs."
            kubectl logs $POD -n $NAMESPACE
            exit 1
        fi
    done
}

# Call the function with the argument "NAMESPACE"
check_pod_logs "servicehubval-operationcontainer-server"
