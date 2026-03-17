#!/bin/bash

export POSTGRES_PASSWORD="Master1234" && kubectl run postgresql-client --rm --tty -i --restart='Never' --namespace go-app --image docker.io/bitnami/postgresql:17.6.0-debian-12-r4 --env="PGPASSWORD=$POSTGRES_PASSWORD" --command -- psql --host postgresql -U postgres -d postgres -p 5432
