#!/bin/bash
rm -rf ./pbs ./postgres
cp -r -p -n /opt/pbs pbs
cp -r -p -n /opt/postgres postgres
cp entryponit.sh init.sh pbs pbs
