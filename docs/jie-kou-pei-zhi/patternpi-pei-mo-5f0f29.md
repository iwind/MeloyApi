# pattern\(匹配模式\)

定义可匹配的路径：

```json
"pattern": "/category/:categoryId/article/:id,
"path": "/category/articleDetail"
```

其中的`:categoryId`和`:id`就在路径中定义了两个变量，所有匹配了这个模式的都会自动解析成类似于以下的URL：

```
/category/articleDetail?categoryId=123&id=456
```



