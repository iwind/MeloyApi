# meloy-api debug - 调试模式

以调试模式启动网关进程：

```
./meloy-api debug
```

启动后，所有的请求信息都会在控制台上被打印出来，类似于：

```
2017/07/17 10:09:09 {
    "Address": "http://meloy.cn/test/post",
    "Api": "/test/post",
    "HasErrors": false,
    "HitCache": false,
    "TimeMs": 25,
    "URI": "/test/post"
}
2017/07/17 10:09:11 {
    "Address": "http://meloy.cn/test/get",
    "Api": "/test/get",
    "Log": "真的很不错",
    "URI": "/test/get"
}
2017/07/17 10:09:11 {
    "Address": "http://meloy.cn/test/get",
    "Api": "/test/get",
    "HasErrors": false,
    "HitCache": false,
    "TimeMs": 2,
    "URI": "/test/get"
}
```



