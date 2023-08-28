#!/usr/bin/env bash

set -e

for pid in $(ps aux | grep "axiom" | awk '{print $2}'); do
  kill -9 $pid 2>/dev/null || continue 
done