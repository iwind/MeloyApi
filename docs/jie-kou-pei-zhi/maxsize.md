# maxSize\(最大请求尺寸\)

定义接口能接受的最大请求尺寸，在上传文件时由此可以限制上传的文件大小：

```json
{
   ...
   "maxSize": "2m"
   ...
}
```

可以使用的单位为：`b（或bytes）`、`k（或kb）`、`m（或mb）`、`g（或gb）`。

如果要想设置全局的限制，可以修改[`Server(API服务器)配置`](/chapter1/serverfu-wu-566829.md)，如果在API和服务器配置中都没有设置`maxSize`，则默认为`32m`。

