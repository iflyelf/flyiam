package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iflyelf/flyiam/internal/config"
	"github.com/iflyelf/flyiam/internal/handler"
	"github.com/iflyelf/flyiam/internal/logic/scheduler"
	"github.com/iflyelf/flyiam/internal/pkg/web"
	"github.com/iflyelf/flyiam/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var (
	configFile  = flag.String("c", "etc/config.yaml", "配置文件路径")
	showVersion = flag.Bool("version", false, "显示版本号并退出")
	Version     = "dev"
	BuildTime   = "unknown"
	GitCommit   = "unknown"
)

func main() {
	flag.Parse()

	// 显示版本号并退出
	if *showVersion {
		fmt.Printf("flyiam version %s (commit %s, built %s)\n", Version, GitCommit, BuildTime)
		return
	}

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 环境变量覆盖（支持容器编排自定义端口/地址等）
	c.ApplyEnvOverrides()

	// 配置日志
	logx.MustSetup(c.Log)

	// 打印启动信息
	printBanner()

	// 创建 REST 服务器
	// 关键：未匹配的请求交给 SPA 处理器，用于提供前端静态文件与前端路由回退
	server := rest.MustNewServer(c.RestConf, rest.WithNotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Web.Embedded {
			web.SPAHandler().ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})))
	defer server.Stop()

	// 初始化服务上下文（数据库、Casdoor、缓存）
	svcCtx := svc.NewServiceContext(c)

	// 启动定时任务调度器
	schedCtx, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	scheduler.New(svcCtx.DB, svcCtx.CasdoorClient).Start(schedCtx)

	// 注册业务路由
	handler.RegisterHandlers(server, svcCtx)

	// 打印服务信息
	printServerInfo(c)

	// 启动服务器
	go func() {
		logx.Info("🚀 启动 FlyIAM 服务器...")
		server.Start()
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logx.Info("正在关闭服务器...")
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Stop()
	logx.Info("服务器已关闭")
}

func printBanner() {
	banner := `
========================================
   FlyIAM - Identity & Access Management
========================================
版本:       %s
构建时间:   %s
Git Commit: %s
========================================
`
	fmt.Printf(banner, Version, BuildTime, GitCommit)
}

func printServerInfo(c config.Config) {
	info := `
🌍 服务信息
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  监听地址:   %s:%d
  运行模式:   %s
  日志级别:   %s
  数据库:     %s:%d/%s
  Redis:      %s:%d (启用: %v)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  🏠 前端页面:  http://localhost:%d
  📊 健康检查:  http://localhost:%d/health
  📚 API 文档:  http://localhost:%d/api
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ 服务器启动成功！按 Ctrl+C 停止服务。
`
	fmt.Printf(info,
		c.Host, c.Port,
		c.Mode,
		c.Log.Level,
		c.Database.Host, c.Database.Port, c.Database.DBName,
		c.Redis.Host, c.Redis.Port, c.Redis.Enabled,
		c.Port, c.Port, c.Port,
	)
}
