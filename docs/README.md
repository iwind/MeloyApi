# MeloyAPI

`MeloyAPI`是使用`GO`语言编写的高性能API网关。

```
                 _________________
Client  <-->    | MeloyAPI Server |     <--> API Server
                |                 |
                | [ Proxy      ]  |
                | [ Statistics ]  |
                | [ Monitor    ]  |
                | [ Cache      ]  |
                 _________________
```

## 为什么我们需要API网关？

在传统的开发过程中，我们维护API的方式是在一个文档中写入每个API的输入参数、输出值等，不仅管理效率低下，而且在客户端调用过程中，对API的响应速度、调用频率等完全一无所知，导致随着业务的增长，API的增多，事情逐渐变得不可控起来。这个时候，API网关就应运而生了，`MeloyAPI`就是其中的一个实现。

## MeloyAPI的目标

`MeloyAPI`有三大目标需要实现：

* 1、简化API管理：
  * 一个命令启动即可使用
  * 配合`MeloyAdmin`实现数据可视化
  * 使用`JSON`定义简化API配置
  * 集成`Git`，让API配置更新更简单
  * 提供一组管理接口，可配合`MeloyAdmin`生成API文档
* 2、优化API性能：
  * 使用缓存、校验等功能降低API请求次数，提升API整体性能
* 3、简化程序开发
  * 支持简单易用的异步调用
  * 未来会集成任务管理（`crontab`） 、锁（`lock`/`unlock`）、消息（`pub`/`sub`）等功能



