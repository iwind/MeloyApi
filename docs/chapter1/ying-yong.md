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

## 限流

可以在应用中使用`limits`配置接口每分钟请求数和每天的请求数：

```json
{
 ...
  "limits": {
    "requests": {
      "minute": 100,
      "day": 100000
    }
  },
  
  ...
}
```

其中`minute`为每分钟最大请求数，`day`为每天的最大请求数，两者可以都设置，也可以都不设置；如果设置为`0`，表示此项不限制。

## 用户验证

可以设置API请求时的用户：

```json
{
  ...
  
  "users": [
    {
        "type": "account",
        "username": "zhangsan",
        "password": "123456"
    },
    ...
  ],
  
  ...
}
```

可以加入一组用户，用户类型目前只支持`account`一种，当设置了用户之后，用户要调用API，必须在请求的Header中加入：

```
Meloy-Username: zhangsan
Meloy-Password: 123456
```

否则会提示`403`权限受限。

