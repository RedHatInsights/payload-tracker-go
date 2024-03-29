#!/bin/bash

./app-common-bash > /tmp/vars.sh
source /tmp/vars.sh

RETENTION_DAYS=${RETENTION_DAYS:-7}
MAX_NUMBER_OF_RETRIES=${MAX_NUMBER_OF_RETRIES:-3}
SLEEP_TIME=${SLEEP_TIME:-10}

echo "RETENTION_DAYS: $RETENTION_DAYS"
echo "MAX_NUMBER_OF_RETRIES: $MAX_NUMBER_OF_RETRIES"
echo "SLEEP_TIME: $SLEEP_TIME"

PGPASSWORD=$CLOWDER_DATABASE_PASSWORD psql -h $CLOWDER_DATABASE_HOSTNAME -U $CLOWDER_DATABASE_USERNAME -d $CLOWDER_DATABASE_NAME -c "DELETE FROM payload_statuses WHERE created_at < (NOW() - interval '$RETENTION_DAYS days');"
PGPASSWORD=$CLOWDER_DATABASE_PASSWORD psql -h $CLOWDER_DATABASE_HOSTNAME -U $CLOWDER_DATABASE_USERNAME -d $CLOWDER_DATABASE_NAME -c "DELETE FROM payloads WHERE created_at < (NOW() - interval '$RETENTION_DAYS days');"
PGPASSWORD=$CLOWDER_DATABASE_PASSWORD psql -h $CLOWDER_DATABASE_HOSTNAME -U $CLOWDER_DATABASE_USERNAME -d $CLOWDER_DATABASE_NAME -c "VACUUM ANALYZE payload_statuses;"
PGPASSWORD=$CLOWDER_DATABASE_PASSWORD psql -h $CLOWDER_DATABASE_HOSTNAME -U $CLOWDER_DATABASE_USERNAME -d $CLOWDER_DATABASE_NAME -c "VACUUM ANALYZE payloads;"

for i in $(seq 1 ${MAX_NUMBER_OF_RETRIES})
do
    echo "Creating partition"

    # Try to create the partition ...if it works, jump out of the retry logic
    PGPASSWORD=$CLOWDER_DATABASE_PASSWORD psql -h $CLOWDER_DATABASE_HOSTNAME -U $CLOWDER_DATABASE_USERNAME -d $CLOWDER_DATABASE_NAME -c "SELECT create_partition(NOW()::DATE + INTERVAL '1 DAY', NOW()::DATE + INTERVAL '2 DAY');" && break || sleep $SLEEP_TIME
done

PGPASSWORD=$CLOWDER_DATABASE_PASSWORD psql -h $CLOWDER_DATABASE_HOSTNAME -U $CLOWDER_DATABASE_USERNAME -d $CLOWDER_DATABASE_NAME -c "SELECT drop_partition(NOW()::DATE - INTERVAL '$RETENTION_DAYS DAY', NOW()::DATE - (($RETENTION_DAYS - 1) || ' DAY')::INTERVAL);"
