#!/bin/bash

exec /playedd \
	--wsAddr="$WS_ADDR" \
	--grpcAddr="$GRPC_ADDR" \
	--redisAddr="$REDIS_ADDR"
