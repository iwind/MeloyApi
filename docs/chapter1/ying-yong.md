# API应用

可以在`config/app.json`中修改API应用的相关配置：

```json
{
  "host": "127.0.0.1",
  "port": 8000,
  "allow": {
    "clients": [
      "127.0.0.1", "192.168.1.1"
    ]
  }  
}
```

其中

* `host`是`meloy-api`启动时绑定的主机地址，可以是`ip`或者是主机名（类似于`localhost`），如果是`0.0.0.0`则表示任何可以访问本主机的IP、主机名等都可以访问
* `port`是`meloy-api`启动时绑定的主机端口
* `allow.clients` - 允许访问的客户端IP，如果不设置此项或者此项为空数组，则表示不限制
* `deny.clients` - 禁止访问的客户端IP，如果不设置此项或者此项为空数组，则表示不限制

在设置好`host`和`port`之后，启动`meloy-api`，就可以在浏览器上访问：

```
http://127.0.0.1:8000
```

所有API也是通过此主机地址访问：

```
http://127.0.0.1:8000/users
http://127.0.0.1:8000/gallery/photos
```



