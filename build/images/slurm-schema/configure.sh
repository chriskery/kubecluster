#!/bin/bash

addUserIfNotExist(){
  if id -u "$1" >/dev/null 2>&1; then
     echo "$1 exists"
  else
     echo "create user $1"
     useradd "$1"
  fi
}
addUserIfNotExist slurm
addUserIfNotExist munge

mkdir -p /etc/munge
# shellcheck disable=SC2024

chown munge:munge /etc/munge/munge.key
chmod 400 /etc/munge/munge.key
mkdir -p /var/lib/munge
mkdir -p /var/run/munge
mkdir -p /var/log/munge
chown -R munge:munge /var/lib/munge
chown -R munge:munge /var/run/munge
chown -R munge:munge /var/log/munge

##cp  relative cmd

# shellcheck disable=SC2162
read -p "please input slurm relative cmd source dir :" cmddir

if [ ! -d "$cmddir" ]; then
   echo "$cmddir not exist, skip cp  slurm relative cmd"
   exit 0;
fi


cp  -p -L "$cmddir"/slurm/bin/* /usr/bin
cp  -p -L "$cmddir"/slurm/sbin/* /usr/sbin
cp  -p -L "$cmddir"/munge/bin/* /usr/bin
cp  -p -L "$cmddir"/munge/sbin/* /usr/sbin
if [ ! -e /etc/munge/munge.key ];then
  cp  -p -L "$cmddir"/munge/munge/munge.key /etc/munge/munge.key
fi
#dd if=/dev/urandom bs=1 count=1024 > /etc/munge/munge.key

# shellcheck disable=SC2024
echo "$cmddir/slurm/lib"  >> /etc/ld.so.conf.d/slurm-x86_64.conf
# shellcheck disable=SC2024
echo "$cmddir/slurm/lib/slurm" >> /etc/ld.so.conf.d/slurm-x86_64.conf
# shellcheck disable=SC2024
echo "$cmddir/munge/lib" >>  /etc/ld.so.conf.d/munge-x86_64.conf

ldconfig

echo "configure completed"