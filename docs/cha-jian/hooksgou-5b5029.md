# meloy-api start

启动API网关：

```bash
./meloy-api start
```

启动后如果`data/`、`logs/`目录不存在，会自动创建这些目录。

启动后会在`data/pid`文件中写入当前启动的网关的进程PID，以便实现对进程的关闭、重启等。

成功启动后，可以在系统中查看此进程：

```
LiuXiangchaos-MacBook-Pro:main root# ps ax|grep meloy-api
87510 s000  S      0:00.01 (meloy-api)
87749 s000  S+     0:00.00 grep meloy-api
```



