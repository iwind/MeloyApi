# response\(返回值定义\)

定义API返回值信息：

```json
"response": {
  "string": "Hello, World"
}
```

定义了`response`后，客户端调用API的返回值恒为`Hello, World`。

除了`string`之外，全部支持的返回类型有：

* `string` - 返回字符串
* `binary` - 返回二进制的Base64Encode数据
* `xml` - 返回XML字符串
* `json` - 返回JSON数据

其中，`json`数据可以如以下定义：

```json
  "params": [],

  "response": {
    "json": {
      "code": 200,
      "message": "OK",
      "data": {
        "ok": true,
        "name": "Li Bai"
      }
    }
  }
```



