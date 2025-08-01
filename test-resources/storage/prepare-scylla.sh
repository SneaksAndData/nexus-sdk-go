#!/usr/bin/env bash

cqlsh localhost -e "CREATE KEYSPACE nexus WITH replication = { 'class': 'SimpleStrategy', 'replication_factor': 1 };"

echo 'Applying checkpoints table'

cqlsh localhost -f /opt/storage/checkpoints.cql

echo 'Checking table'

cqlsh localhost -e 'SELECT * FROM nexus.checkpoints'

echo 'Applying submission_buffer table'

cqlsh localhost -f /opt/storage/submission_buffer.cql

echo 'Checking table'

cqlsh localhost -e 'SELECT * FROM nexus.submission_buffer'
