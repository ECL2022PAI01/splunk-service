#!/bin/bash

# Get the public IP of a cluster node
public_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')

# Get the NodePort assigned to your service
node_port=$(kubectl get svc temporary-lb-service-gitea -n gitea -o jsonpath='{.spec.ports[0].nodePort}')

# Set the environment variables
export CLUSTER_PUBLIC_IP=$public_ip
export SERVICE_NODE_PORT=$node_port

export GITEA_ENDPOINT_TOKEN="http://$public_ip:$node_port"
# Print the values for verification
echo "Cluster Public IP: $GITEA_ENDPOINT_TOKEN"
echo "Service NodePort: $node_port"
