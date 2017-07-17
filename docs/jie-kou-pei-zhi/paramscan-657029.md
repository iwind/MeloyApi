# params\(参数\)

定义API可接收的参数，一般由名称（`name`）、类型（`type`）、 描述（`description`）组成：

```json
{
  ...
  "params": [
	{
	  "name": "mobile",
	  "type": "string",
	  "description": "手机号"
	},
	{
	  "name": "code",
	  "type": "int",
	  "description": "验证码"
	},
	{
	  "name": "token",
	  "type": "string",
	  "description": "校验用的令牌"
	}
  ]
}
```



