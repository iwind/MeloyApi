# 管理API

`MeloyAPI`提供一组管理API的API，方便开发者操作API。

## 格式化

在请求API的时候，如果加入参数`_pretty=true`，输出的JSON就是格式化后的，比如：

```
http://localhost:8001/@monitor?_pretty=true
```

输出的JSON为：

```json
{
  "code": 200,
  "data": {
    "cost": 75,
    "errorsPercent": 0,
    "hitsPercent": 5,
    "load15m": "2.03",
    "load1m": "1.75",
    "load5m": "1.94",
    "memory": 9967864,
    "requestsPerMin": 4,
    "routines": 15
  },
  "message": "Success"
}
```



