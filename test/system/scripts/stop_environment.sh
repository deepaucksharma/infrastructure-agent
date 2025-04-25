#!/bin/bash
# Script to stop the system test environment for infrastructure agent tests

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TEST_DIR="$SCRIPT_DIR/.."

echo "Stopping and removing test environment containers..."
docker-compose -f "$TEST_DIR/docker-compose.yml" down

echo "Test environment stopped."