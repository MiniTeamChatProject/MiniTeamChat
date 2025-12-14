# 🏆 聊天室项目 - 用户微服务 (User Microservice)

> 这是一个基于 Go (Gin) + PostgreSQL 的用户鉴权微服务，提供注册、登录及用户信息管理功能。

## 📂 项目结构
- `main.go`: 核心源代码
- `go.mod` / `go.sum`: 依赖配置文件
- `run.sh`: Linux 一键启动脚本

## 🚀 快速开始 (How to Run)

### 前置要求
1. 安装 **Go** (1.20+)
2. 安装 **PostgreSQL** 并创建数据库 `benutzer_db`

### 启动命令
在 Linux 终端运行以下命令：
```bash
chmod +x ./run.sh
./run.sh