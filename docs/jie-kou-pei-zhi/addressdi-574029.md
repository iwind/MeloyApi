# address\(地址\)\(必填项\)

定义API地址，一般由`SERVER`+`PATH`组成：

```json
"address": "%{server.meloy}/article",
```

也可以使用`api.path`变量：

```json
"address": "%{server.meloy}%{api.path}",
```

其中`meloy`是API服务器配置的代号。

