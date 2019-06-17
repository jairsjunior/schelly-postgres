#!/bin/bash
set +e
# set +x

echo "Starting Postgres API..."
schelly-postgres \
    --listen-ip=$LISTEN_IP \
    --listen-port=$LISTEN_PORT \
    --log-level=$LOG_LEVEL \
    --pre-post-timeout=$PRE_POST_TIMEOUT \
    --pre-backup-command="$PRE_BACKUP_COMMAND" \
    --post-backup-command="$POST_BACKUP_COMMAND" \
    --dbname="$DATABASE_NAME" \
    --host="$DATABASE_CONNECTION_HOST" \
    --port="$DATABASE_CONNECTION_PORT" \
    --username="$DATABASE_AUTH_USERNAME" \
    --password="$DATABASE_AUTH_PASSWORD" \
    --azure-storage="$USE_AZURE_STORAGE" \
    --account-name="$AZURE_STORAGE_ACCOUNT_NAME" \
    --account-key="$AZURE_STORAGE_ACCOUNT_KEY" \
    --container-name="$AZURE_STORAGE_CONTAINER_NAME" \

