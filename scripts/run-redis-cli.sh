#!/bin/bash

REDIS_PASSWORD=$(kubectl get secret --namespace go-app casbin-redis -o jsonpath="{.data.redis-password}" | base64 -d)
kubectl run --namespace go-app redis-client --restart='Never' --env REDIS_PASSWORD=$REDIS_PASSWORD --image registry-1.docker.io/bitnami/redis:latest --command -- sleep infinity
kubectl exec --tty -i redis-client --namespace go-app -- bash
