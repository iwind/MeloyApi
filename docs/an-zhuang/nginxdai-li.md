# Nginx代理

可以使用`nginx`作为`MeloyAPI`的代理程序，来隐藏`MeloyAPI`的端口，一个典型的配置如下：

```nginx
server {
    listen 80;
    server_name  meloy.cn;

    client_max_body_size 128M;

    location / {
        proxy_set_header Host $host;
        proxy_pass http://192.168.1.100:8000;
    }

    location ~ ^/@mock {
        proxy_set_header Host $host;
        proxy_pass http://192.168.1.100:8001;
    }
}
```

其中`meloy.cn`是我们使用的域名，`8000`是网关API端口，`8001`是网关管理端口，同时也提供模拟数据；`192.168.1.100`是网关程序所在的服务器的IP。

把此配置放到`nginx.conf`中，重载`nginx`服务，就可以使用`http://meloy.cn/API路径`来访问API，使用`http://meloy.cn/@mock/API路径`来访问模拟数据了。

