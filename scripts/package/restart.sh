#!/usr/bin/env bash
set -e

kill `cat ./axiom.pid`
sleep 2
mv axiom axiom.bak
# $1 is new axiom path
cp $1 . 
bash nohup start.sh > /dev/null 2 > &1 &