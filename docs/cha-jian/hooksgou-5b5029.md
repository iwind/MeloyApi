# Hook\(钩子\)

可以在应用启动之前加入自己的钩子函数：

```go
hook1 := MeloyApi.Hook{
    BeforeFunc: func(context *MeloyApi.HookContext, next func()) {
        next();
    },
    AfterFunc: func(context *MeloyApi.HookContext) {

    },
}
MeloyApi.GetHookManager().AddHook()

//启用应用
MeloyApi.Start(appDir)
```

其中`BeforeFunc()`在API请求开始之前调用，如果在`BeforeFunc()`中不调用`next()`，则下面的其他钩子函数和API请求将不会被执行；`AfterFunc()`在API请求结束之后调用。

可以加入多个钩子函数：

```go
MeloyApi.GetHookManager().AddHook(hook1)
MeloyApi.GetHookManager().AddHook(hook2)
MeloyApi.GetHookManager().AddHook(hook3)

//启用应用
MeloyApi.Start(appDir)
```

钩子和API的调用顺序是：

```
hook1 BeforeFunc()
hook2 BeforeFunc()
hook3 BeforeFunc()
API Request
hook3 AfterFunc()
hook2 AfterFunc()
hook1 AfterFunc()
```

在`MeloyApi.HookContext`中可以读取和操作`http.ResponseWriter(对调用端的响应写入器)`和 `http.Request（API请求对象）`、`http.Response（API响应对象）`:

```go
type HookContext struct {
    Writer http.ResponseWriter
    Request *http.Request
    Api *Api
    Response *http.Response
    Error error
}
```



