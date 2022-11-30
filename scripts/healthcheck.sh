#!/usr/bin/env sh
gateway_port=$(grep -m 1 gateway </root/.bitxhub/bitxhub.toml | awk '{print $3}')
status=$(bitxhub client --gateway http://localhost:"$gateway_port"/v1 chain status)
if [ "$status" = "normal" ]; then
  exit 0
else
  exit 1
fi
