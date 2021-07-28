# 节点日志查看

这里主要介绍如何查看跨链系统中bitxhub和pier节点的日志信息。

### BitXHub

如果是在终端前台启动的节点，那么日志会实时打印在终端上，观察其无报错即可；

如果是通过nohup等后台启动的节点，在节点主配置目录的logs文件夹中就是节点的日志文件，打开即可检查日志，一般情况下除了出块，bitxhub节点之间会定时相互`ping`其它节点并返回延时信息，可以简单看到节点集群之间的网络状态。

<img src="../../../assets/bitxhub-log.png" alt="bitxhub-log" style="zoom:50%;" />

### Pier

如果是在终端前台启动的节点，那么日志会实时打印在终端上，观察其无报错即可；

如果是通过nohup等后台启动的节点，在节点主配置目录的logs文件夹中就是节点的日志文件，打开即可检查日志，

<img src="../../../assets/pier-log.png" alt="pier-log" style="zoom:50%;" />



