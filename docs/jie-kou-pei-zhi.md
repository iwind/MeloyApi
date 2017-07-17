# API配置

## 配置文件

API配置放在`apis/`目录中，且可以创建多级目录：

```
apis/
  api1.json
  api2.json
  新目录1/
      api3.json
      api4.json
  ...
```

## mock\(模拟数据\)

每一个API都可以设置一个模拟数据文件，只需要把配置的扩展名改成 `mock.json`：

```
apis/
  api1.json
  api1.mock.json
  api2.json
  api2.mock.json
  新目录1/
      api3.json
      api3.mock.json
      api4.json
      api4.mock.json
  ...
```

`mock.json`中的数据理论上可以是任意格式的数据，并不限于`json`。

## 配置选项

一个API中至少必须定义`path`、`address`、`methods`三个选项，然后一般也配置`description`和`params`，以下是一个典型的API :

```json
{
  "path": "/article",
  "address": "%{server.meloy}%{api.path}",
  "description": "文章详情",
  "methods": [ "get", "post" ],
  "params": [
    {
      "name": "articleId",
      "type": "int",
      "description": "公告、文章ID"
    }
  ]
}
```

注意其中的`%{api.path}`指的是当前API定义的`path`，可以改成：

```json
...
"address": "%{server.meloy}/article",
...
```



