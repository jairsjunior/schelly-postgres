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
    --post-backup-command="$POST_BACKUP_COMMAND"
    --file="$BACKUP_FILE_PATH" \
