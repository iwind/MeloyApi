# 调试指令

可以在某个API中通过`Meloy-Api-DebugXXX`来设置调试信息：

```php
header("Meloy-Api-Debug1: 调试信息1");
header("Meloy-Api-Debug2: 调试信息2");
```

当该API被调用后，通过`/@api/[:path]/debug/logs`就可以读取该接口设置的所有调试信息。

