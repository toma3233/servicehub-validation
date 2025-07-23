#!/bin/bash

# Usage: ./checkServiceStatus.sh <namespace1> [<namespace2> ...]
# Checks the status of all services in the given namespaces and ensures they are not in 'Pending' state.
# Exits with error if any service is not assigned an external IP (for LoadBalancer type) or is not ready.


# Default to app-routing-system if no namespaces are provided
if [ "$#" -lt 1 ]; then
  NAMESPACES=("app-routing-system")
  echo "No namespaces specified. Defaulting to: ${NAMESPACES[*]}"
else
  NAMESPACES=("$@")
fi

# Wait time for service readiness (default 60s, can override with SERVICE_WAIT_TIME env var)
SERVICE_WAIT_TIME="${SERVICE_WAIT_TIME:-60s}"

for NAMESPACE in "${NAMESPACES[@]}"; do
  echo "Checking services in namespace: $NAMESPACE"
  # Wait for namespace to exist and be active
  kubectl wait --for=jsonpath='{.status.phase}=Active' --timeout="$SERVICE_WAIT_TIME" namespace/$NAMESPACE >/dev/null
  if [ $? -ne 0 ]; then
    echo "ERROR: $NAMESPACE does not exist or is not Active after waiting $SERVICE_WAIT_TIME."
    exit 1
  fi
  SERVICES=$(kubectl get svc -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $1}')
  if [ -z "$SERVICES" ]; then
    echo "ERROR: No services found in namespace $NAMESPACE."
    exit 1
  fi
  for SERVICE in $SERVICES; do
    # Service objects are typically ready immediately, no wait needed
    TYPE=$(kubectl get svc "$SERVICE" -n "$NAMESPACE" -o jsonpath='{.spec.type}')
    STATUS_MSG="Service $SERVICE in $NAMESPACE is OK."
    if [ "$TYPE" = "LoadBalancer" ]; then
      # Wait for external IP to be assigned, up to SERVICE_WAIT_TIME
      kubectl wait --for=jsonpath='{.status.loadBalancer.ingress[0].ip}' --timeout="$SERVICE_WAIT_TIME" svc "$SERVICE" -n "$NAMESPACE" >/dev/null
      if [ $? -ne 0 ]; then
        STATUS_MSG="ERROR: Service $SERVICE in $NAMESPACE is of type LoadBalancer but has no external IP assigned (Pending) after waiting $SERVICE_WAIT_TIME."
        echo "$STATUS_MSG"
        exit 1
      fi
    fi
  done
  echo "All services in $NAMESPACE are OK."
done

echo "All checked services are healthy."
