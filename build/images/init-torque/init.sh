#!/bin/bash
cp /opt/pbs/pbs /etc/init.d/pbs
cp /opt/pbs/pbs.conf /etc/pbs.conf
if [ ! -e /etc/ld.so.conf.d/pbs.conf ];then
        echo "/opt/pbs/lib" >> /etc/ld.so.conf.d/pbs.conf
fi;
if [ ! -e /etc/ld.so.conf.d/postgres.conf ];then
        echo "/opt/postgre/lib" >> /etc/ld.so.conf.d/postgres.conf
fi;
cp -r -n /opt/pbs/bin/* /usr/bin
cp -r -n /opt/pbs/sbin/* /usr/sbin
cp -r -n /opt/postgres/bin/* /usr/bin
cp -r -n /opt/postgres/lib/* /lib
ldconfig

useradd postgres
if [ ! -d /home/postgres ];then
        mkdir -p /home/postgres
        chown postgres:postgres /home/postgres
fi

cp -r -n /opt/postgres/share/* /usr/share

pbs_conf_file=/etc/pbs.conf
mom_conf_file=/var/spool/pbs/mom_priv/config
hostname=$(hostname)

# replace hostname in pbs.conf and mom_priv/config
sed -i "s/PBS_SERVER=.*/PBS_SERVER=$hostname/" $pbs_conf_file
sed -i "s/\$clienthost .*/\$clienthost $hostname/" $mom_conf_file

# start PBS Pro
/etc/init.d/pbs start

# create default non-root user
