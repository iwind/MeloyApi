# /@monitor

取得监控信息，包括负载、内存等，示例返回：{

```json
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

返回字段说明：

| 字段代号 | 字段类型 | 字段说明 |
| :--- | :--- | :--- |
| cost | int | 平均耗时（ms） |
| errorsPercent | float | 错误率 |
| hitsPercent | float | 缓存命中率 |
| requestsPerMin | int | 平均每分钟的请求数 |
| load1m | string | 1分钟的负载 |
| load5m | string | 5分钟的负载 |
| load15m | string | 15分钟的负载 |
| memory | int | 内存（字节） |
| routines | int | GoRoutine数量 |



