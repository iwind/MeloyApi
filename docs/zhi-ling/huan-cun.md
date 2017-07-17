# 缓存指令

可以在接口中通过HEADER信息控制`MeloyAPI`的行为。

## 缓存键值（Key）

`MeloyAPI`自动把请求的URI当做缓存的键，所以不需要再次设置。

## 缓存标签（Tag）

每个缓存都可以设置一组标签，用于将来删除跟此标签相关联的所有缓存，当然也可以不设置任何标签。

标签使用`Meloy-Api-Cache-TagXXX`来设置，以下都是合法的标签：

```
Meloy-Api-Cache-TagUser
Meloy-Api-Cache-TagProduct
```

## 缓存时间

可以使用`Meloy-Api-Cache-Life-Ms`设置缓存时间，单位是`ms`（0.001秒）。

## 设置缓存示例

```php
header("Meloy-Api-Cache-TagProduct:product.100");
header("Meloy-Api-Cache-TagUser:user.123");
header("Meloy-Api-Cache-Life-Ms:3600000");
```

## 清除缓存

可以使用`Meloy-Api-Cache-DeleteXXX`删除某个缓存标签对应的所有缓存：

```php
header("Meloy-Api-Cache-DeleteUser:user:123");
header("Meloy-Api-Cache-DeleteProduct:product:100");
```

## 缓存管理API

`MeloyAPI`提供了一组缓存管理的API：

```
/@cache/clear             # 清除API所有缓存
/@cache/[path]/clear         # 清除所有缓存
/@cache/tag                 # 获取TAG信息
/@cache/tag/:tag/deelte     # 清除跟TAG相关的的缓存
```



