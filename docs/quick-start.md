## Create a Torque Cluster

**Create Torque YAML**

```
kubectl create -f ../manifests/samples/torque-centos.yaml
```

The torque centos example create a torque cluster with 1 server and 1 worker, 
so it will create two pods to simulate two nodes for the torque cluster

**Get Torque Status**

Execute the following command:
```
kubectl get kubeclusters
```
The output is like:
```shell
> kubectl get kubeclusters
NAME                   AGE   STATE
torque-centos-sample   3s    Running
```

Now you can enter the " server node " and use this torque-centos-sample look like you're actually using a physical torque cluster
```
> kubectl get pods        
NAME                                READY   STATUS    RESTARTS   AGE
nginx-deployment-5bc4c45dc9-npwxp   1/1     Running   16         46h
torque-centos-sample-cpu-0          1/1     Running   0          2m43s
torque-centos-sample-server-0       1/1     Running   0          2m43s
```
torque-centos-sample-server-0 is the server node of cluster torque-centos-sample
```
> kubectl exec -it torque-centos-sample-server-0 /bin/bash                                
kubectl exec [POD] [COMMAND] is DEPRECATED and will be removed in a future version. Use kubectl exec [POD] -- [COMMAND] instead.
[root@torque-centos-sample-server-0 /]#
```

**Using Torque Cluster**
Viewing Nodes' status of torque-centos-sample
```
[root@torque-centos-sample-server-0 pbs]# pbsnodes -a
torque-centos-sample-server-0
     Mom = torque-centos-sample-server-0
     Port = 15002
     pbs_version = 19.0.0
     ntype = PBS
     state = free
     pcpus = 16
     resources_available.arch = linux
     resources_available.host = torque-centos-sample-server-0
     resources_available.mem = 64756484kb
     resources_available.ncpus = 16
     resources_available.vnode = torque-centos-sample-server-0
     resources_assigned.accelerator_memory = 0kb
     resources_assigned.hbmem = 0kb
     resources_assigned.mem = 0kb
     resources_assigned.naccelerators = 0
     resources_assigned.ncpus = 0
     resources_assigned.vmem = 0kb
     resv_enable = True
     sharing = default_shared
     last_state_change_time = Thu Sep 28 07:05:43 2023

torque-centos-sample-cpu-0
     Mom = 10-244-0-56.torque-centos-sample-cpu-0.default.svc.cluster.local
     Port = 15002
     pbs_version = 19.0.0
     ntype = PBS
     state = free
     pcpus = 16
     resources_available.arch = linux
     resources_available.host = 10-244-0-56
     resources_available.mem = 64756484kb
     resources_available.ncpus = 16
     resources_available.vnode = torque-centos-sample-cpu-0
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


