#!/bin/bash

TEST_RESULT=0
DOCKERFILE='Dockerfile-test'
IMAGE='tracker'
POSTGRES_IMAGE='quay.io/cloudservices/postgresql-rds:15-1'
TEARDOWN_RAN=0


teardown() {

    [ "$TEARDOWN_RAN" -ne "0" ] && return

    echo "Running teardown..."

    docker rm -f "$TEST_CONTAINER_NAME"

    # remove postgres container
    docker rm -f postgres
    TEARDOWN_RAN=1
}

trap teardown EXIT ERR SIGINT SIGTERM

mkdir -p artifacts

get_N_chars_commit_hash() {

    local CHARS=${1:-7}

    git rev-parse --short="$CHARS" HEAD
}

TEST_CONTAINER_NAME="tracker-$(get_N_chars_commit_hash 7)"

echo "Pulling postgres image"
docker pull $POSTGRES_IMAGE

echo -e "\n---------------------------------------------------------------\n"

echo "Starting postgres container"
docker run --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5 -d --rm --name postgres -e POSTGRESQL_PASSWORD=crc -e POSTGRESQL_USER=crc -e POSTGRESQL_DATABASE=crc -h 0.0.0.0:5432:5432 $POSTGRES_IMAGE
sleep 5

echo -e "\n---------------------------------------------------------------\n"

echo "Building image"
docker build -f "$DOCKERFILE" -t "$TEST_CONTAINER_NAME" .

echo -e "\n---------------------------------------------------------------\n"

echo "Running container"
docker run -d --rm --name "$TEST_CONTAINER_NAME" "$IMAGE" sleep infinity

echo -e "\n---------------------------------------------------------------\n"

echo "Migrating database"
docker exec --workdir /workdir "$TEST_CONTAINER_NAME"  make run-migration -B -e DB_HOST="localhost" > 'artifacts/migration_logs.txt'
MIGRATION_RESULT=$?

cat artifacts/migration_logs.txt

if [ $MIGRATION_RESULT -eq 0 ]; then
    echo "Migration ran successfully"
else
    echo "Migration failed..."
    sh "exit 1"
    # stop script execution
    exit 1
fi

echo -e "\n---------------------------------------------------------------\n"
echo "Running tests"
docker exec --workdir /workdir -e PATH=/opt/app-root/src/go/bin:$PATH "$TEST_CONTAINER_NAME" make test -e DB_HOST="localhost" > 'artifacts/test_logs.txt'
TEST_RESULT=$?

cat artifacts/test_logs.txt

echo -e "\n---------------------------------------------------------------\n"

if [ $TEST_RESULT -eq 0 ]; then
    echo "Tests ran successfully"
else
    echo "Tests failed..."
    sh "exit 1"
fi
