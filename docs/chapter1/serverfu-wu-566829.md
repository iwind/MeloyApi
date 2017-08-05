# API服务器

可以在`config/servers.json`中配置API服务器：

```json
[
  // Baidu Server
  {
    "code": "baidu",
    "hosts": [
      {
        "address": "https://baidu.com",
        "weight": 10
      }
    ]
  },

  // Meloy Server
  {
    "code": "meloy",

    "request": {
      "maxSize": "10m",
      "timeout": "30s"
    },

    "hosts": [
      {
        "address": "http://api1.meloy.cn",
        "weight": 10
      },
      {
        "address": "http://api2.meloy.cn",
        "weight": 10
      },
      {
        "address": "http://api3.meloy.cn",
        "weight": 20
      }
    ]
  }
]
```

可以配置多个`Server`，每个`Server`设置一个`code`代号，以便于在API定义中使用。

## 主机

每个`Server`可以配置一到多个主机（`host`），并可设置主机的权重：

```json
{
  "address": "http://api1.meloy.cn",
  "weight": 10
},
...
```

每个`host`的`address`是构造API的URL的前缀，比如一个这个定义的API：

```json
{
  "path": "/test/get",
  "address": "%{server.meloy}/test/get",
  "methods": [ "get" ],
  "description": "测试GET请求"
}
```

解析后的API的备选URL就是：

```
http://api1.meloy.cn/test/get
http://api2.meloy.cn/test/get
http://api3.meloy.cn/test/get
```

## 请求配置

可以在服务器设置中设置单个API请求的超时时间（`timeout`）和最大请求尺寸（`maxSize`）：

```json
{
  "code": "meloy",

  "request": {
      "maxSize": "10m",
      "timeout": "30s"
   },

   ...     
}
```

这两个参数每个API也可以单独设置，具体看`API配置`一节中说明。

