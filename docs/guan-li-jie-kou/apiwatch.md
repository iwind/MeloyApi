# /@api/watch

监控最新请求，示例返回：

~~~json
{
  "code": 200,
  "data": [
    {
      "id": 1501379084206425693,
      "createdAt": 1501379084,
      "request": {
        "uri": "/test/delete",
        "method": "GET",
        "data": "GET /test/delete HTTP/1.1\r\nHost: localhost:8000..."
      },
      "response": {
        "statusCode": 200,
        "status": "200 OK",
        "data": "HTTP/1.1 200 OK\r\nDate: Sun, 30 Jul 2017 01:44:44 GMT..."
      }
    },
    {
      "id": 1501378970280148752,
      "createdAt": 1501378970,
      "request": {
        "uri": "/test/put",
        "method": "GET",
        "data": "GET /test/put HTTP/1.1\r\nHost: localhost:8000..."
      },
      "response": {
        "statusCode": 200,
        "status": "200 OK",
        "data": "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked..."
      }
    },
    {
      "id": 1501378963223491901,
      "createdAt": 1501378963,
      "request": {
        "uri": "/test/get",
        "method": "GET",
        "data": "GET /test/get HTTP/1.1\r\nHost: localhost:8000..."
      },
      "response": {
        "statusCode": 200,
        "status": "200 OK",
        "data": "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked..."
      }
    }
  ],
  "message": "Success"
}
~~~

