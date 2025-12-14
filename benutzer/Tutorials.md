# 📘 用户微服务 API 接口手册 (v1.0)

基础信息：

协议：HTTP

数据格式：JSON

基础 URL (本地)：http://localhost:8080

基础 URL (远程)：http://<你的服务器IP>:8080 (部署后使用这个)

## 1. 用户注册 (Register)

* 调用名称/功能：创建新的用户账号。

* 接口地址：/register

* 请求方式：POST

* 请求头 (Headers)：

* Content-Type: application/json

### 请求参数 (Request Body)

字段名	类型	必填	说明

username	string	✅ 是	唯一的登录账号 (如 "tom")

password	string	✅ 是	明文密码 (如 "123456")

nickname	string	❌ 否	聊天室显示的昵称

请求示例 (JSON):

code

JSON

{

  "username": "zhangsan",

  "password": "my_secure_password",

  "nickname": "法外狂徒"

}

## 响应结果 (Response)

成功 (HTTP 200 OK):

code

JSON

{

  "msg": "注册成功",

  "data": {

    "id": 10,

    "username": "zhangsan",

    "nickname": "法外狂徒",

    "created_at": "2025-12-14T..."

  }

}

失败 (HTTP 400 Bad Request):

code

JSON

{

  "error": "用户名已存在"

}

### 2. 用户登录 (Login)

调用名称/功能：验证身份并获取访问令牌 (Token)。

接口地址：/login

请求方式：POST

请求头 (Headers)：

Content-Type: application/json

### 请求参数 (Request Body)

字段名	类型	必填	说明

username	string	✅ 是	注册时的账号

password	string	✅ 是	注册时的密码

请求示例 (JSON):

code

JSON

{

  "username": "zhangsan",

  "password": "my_secure_password"

}

### 响应结果 (Response)

成功 (HTTP 200 OK):

code

JSON

{

  "msg": "登录成功",

  "token": "eyJhbGciOiJIUzI1NiIs...",  // <--- 极其重要！前端需保存此Token

  "user": {

    "id": 10,

    "username": "zhangsan",

    "nickname": "法外狂徒",

    "created_at": "2025-12-14T..."

  }

}

### 失败 (HTTP 401 Unauthorized):

code

JSON

{

  "error": "密码错误" 

  // 或者 "用户不存在"

}
## 3. [预留] 获取用户信息 (Get Profile)

这是下一步计划要实现的接口，暂时先列在这里作为占位符。


调用名称/功能：使用 Token 换取用户详情。

接口地址：/user/profile (暂定)

请求方式：GET

请求头 (Headers)：

Authorization: Bearer <你的Token> (必须携带登录时拿到的 Token)

### 请求参数

无 (通过 Token 识别身份)

### 响应结果 (Response)

成功 (HTTP 200 OK):

code

JSON

{

  "id": 10,

  "username": "zhangsan",

  "nickname": "法外狂徒"

}

## 给开发者的备注 (Developer Notes)

关于密码：前端在发送请求时，直接发送明文密码即可。后端的 /register 接口会自动进行 Bcrypt 加密，/
login 接口会自动进行哈希比对。

关于 Token：

Token 是用户的身份证。

前端登录成功后，应该把 token 字段保存在浏览器的 localStorage 或 App 的存储中。

以后前端要请求“修改头像”、“发送消息”等接口时，必须把这个 Token 带上，否则后端会拒绝服务。

如何测试：

可以使用命令行工具 curl (如你之前所做)。

也可以使用图形化工具 Postman (推荐下载一个，测试 API 非常方便)。
