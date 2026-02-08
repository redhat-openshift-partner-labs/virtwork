#!/bin/sh
set -e

if [ -n "$VIRTWORK_COMMAND" ]; then
    exec /usr/local/bin/virtwork $VIRTWORK_COMMAND $VIRTWORK_ARGS
else
    echo "No VIRTWORK_COMMAND set. Pod will sleep. Use 'oc exec' to run virtwork manually."
    exec sleep infinity
fi
