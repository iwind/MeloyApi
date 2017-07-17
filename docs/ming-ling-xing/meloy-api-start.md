# meloy-api start

启动网关：

```bash
./meloy-api start
```

启动后，如果`data/`、`logs/`目录不存在，则会自动创建，并在`data/pid`文件中写入启动的网关进程PID。

成功启动后，可以通过`ps`命令查看启动的进程：

```
LiuXiangchaos-MacBook-Pro:main root# ps ax|grep meloy-api
87510 s000  S      0:00.01 (meloy-api)
87749 s000  S+     0:00.00 grep meloy-api
```



