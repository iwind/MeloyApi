# isAsynchronous\(是否异步\)

定义接口是否可异步调用，默认为`false`，如果设置为`true`，则表示`MeloyAPI`无需等待API返回数据，即可向调用的客户端返回数据：

```json
"isAsynchronous": false
```

如果设置为`true`，可以配合`response`选项设置API调用后返回的内容。

