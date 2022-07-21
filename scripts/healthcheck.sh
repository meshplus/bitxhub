#!/usr/bin/env bash
status=$(bitxhub client --gateway http://localhost:${GateWayPort}/v1 chain status)
if [ $status = "normal" ]; then
    exit 0;
else
    exit 1;
fi