# mysqlgogogo - a mysql compatible database 

我现在不知道我要做什么东西，代码也是随心所欲。

这个repo也仅仅用来中转。

公司想到写两行，家里想到写两行，代码规范更是没有。

至于什么时候，完成多大的项目，则是完全没有规划。

### 基本目标

1. 不支持utf8之外的编码。
2. 简单的sql支持。


### 高级目标

1. mvcc
2. 查询计划


### 说明

mysql协议部分来自kingshard，解析器来自pingcap/tidb，存储引擎来自goleveldb。