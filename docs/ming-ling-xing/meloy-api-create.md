# meloy-api create \[API Code\]

创建新的API，会在`apis/`目录下自动生成一个API模板，比如：

```shell
./meloy-api create hello_world
```

程序会自动创建`apis/hello_world.json`，并填入以下内容：

```json
{
  "path": "/hello/world",
  "name": "接口名称",
  "description": "接口描述",
  "address": "%{server.代号}%{api.path}",
  "methods": [ "get", "post" ],
  "params": [
    {
      "name": "",
      "type": "",
      "description": ""
    }
  ]
}
```

这时候只需要修改`address`中的`server`代号即可。

