## Build a pbspro Cluster

**Create pbspro YAML**

```
kubectl create -f ../manifests/samples/pbspro-centos.yaml
```

The pbspro centos example create a pbspro cluster with 1 server and 1 worker, 
so it will create two pods to simulate two nodes for the pbspro cluster

**Get kubeclusters Status**

Execute the following command:
```
kubectl get kubeclusters
```
The output is like:
```shell
> kubectl get kubeclusters
NAME                   AGE   STATE
pbspro-centos-sample   3s    Running
```

Now you can enter the " server node " as  you're actually using a physical pbspro cluster
```
> kubectl get pods        
NAME                                READY   STATUS    RESTARTS   AGE
nginx-deployment-5bc4c45dc9-npwxp   1/1     Running   16         46h
pbspro-centos-sample-cpu-0          1/1     Running   0          2m43s
pbspro-centos-sample-server-0       1/1     Running   0          2m43s
```
pbspro-centos-sample-server-0 is the server node of cluster pbspro-centos-sample
```
> kubectl exec -it pbspro-centos-sample-server-0 /bin/bash                                
kubectl exec [POD] [COMMAND] is DEPRECATED and will be removed in a future version. Use kubectl exec [POD] -- [COMMAND] instead.
[root@pbspro-centos-sample-server-0 /]#
```

**Using pbspro Cluster**

Viewing Nodes' status of pbspro-centos-sample
```
[root@pbspro-centos-sample-server-0 pbs]# pbsnodes -a
pbspro-centos-sample-server-0
     Mom = pbspro-centos-sample-server-0
     Port = 15002
     pbs_version = 19.0.0
     ntype = PBS
     state = free
     pcpus = 16
     resources_available.arch = linux
     resources_available.host = pbspro-centos-sample-server-0
     resources_available.mem = 64756484kb
     resources_available.ncpus = 16
     resources_available.vnode = pbspro-centos-sample-server-0
     resources_assigned.accelerator_memory = 0kb
     resources_assigned.hbmem = 0kb
     resources_assigned.mem = 0kb
     resources_assigned.naccelerators = 0
     resources_assigned.ncpus = 0
     resources_assigned.vmem = 0kb
     resv_enable = True
     sharing = default_shared
     last_state_change_time = Thu Sep 28 07:05:43 2023

pbspro-centos-sample-cpu-0
     Mom = 10-244-0-56.pbspro-centos-sample-cpu-0.default.svc.cluster.local
     Port = 15002
     pbs_version = 19.0.0
     ntype = PBS
     state = free
     pcpus = 16
     resources_available.arch = linux
     resources_available.host = 10-244-0-56
     resources_available.mem = 64756484kb
     resources_available.ncpus = 16
     resources_available.vnode = pbspro-centos-sample-cpu-0
     resources_assigned.accelerator_memory = 0kb
     resources_assigned.hbmem = 0kb
     resources_assigned.mem = 0kb
     resources_assigned.naccelerators = 0
     resources_assigned.ncpus = 0
     resources_assigned.vmem = 0kb
     resv_enable = True
     sharing = default_shared
     last_state_change_time = Thu Sep 28 07:05:43 2023
```
Switch to the normal user and submit the cluster using [qsub](https://www.jlab.org/hpc/PBS/qsub.html)
```shell
[root@pbspro-centos-sample-server-0 /]# useradd pbsexample
[root@pbspro-centos-sample-server-0 /]# su pbsexample
[pbsexample@pbspro-centos-sample-server-0 /]$ qsub -- hostname
2.pbspro-centos-sample-server-0
[pbsexample@pbspro-centos-sample-server-0 /]$
```
Use [qstat](https://docs.adaptivecomputing.com/torque/4-0-2/Content/topics/commands/qstat.htm) to view the cluster we just submitted
```
[pbsexample@pbspro-centos-sample-server-0 /]$ qstat -a

pbspro-centos-sample-server-0: 
                                                            Req'd  Req'd   Elap
cluster ID          Username Queue    clustername    SessID NDS TSK Memory Time  S Time
--------------- -------- -------- ---------- ------ --- --- ------ ----- - -----
2.pbspro-centos pbsexamp workq    STDIN        1377   1   1    --    --  E 00:00
```


