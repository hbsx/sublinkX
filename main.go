package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sublink/middlewares"
	"sublink/models"
	"sublink/routers"
	"sublink/settings"
	"sublink/utils"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed static/js/*
//go:embed static/css/*
//go:embed static/img/*
//go:embed static/*
var embeddedFiles embed.FS

//go:embed template
var Template embed.FS

// 版本号
var version string

func Templateinit() {
	// 设置template路径
	// 检查目录是否创建
	subFS, err := fs.Sub(Template, "template")
	if err != nil {
		log.Println(err)
		return // 如果出错，直接返回
	}
	entries, err := fs.ReadDir(subFS, ".")
	if err != nil {
		log.Println(err)
		return // 如果出错，直接返回
	}
	// 创建 template 目录。目录需要执行权限才能访问其中的文件。
	if err = os.MkdirAll("./template", 0755); err != nil {
		log.Println(err)
		return
	}
	// 写入默认模板
	for _, entry := range entries {
		_, err := os.Stat("./template/" + entry.Name())
		//如果文件不存在则写入默认模板
		if os.IsNotExist(err) {
			data, err := fs.ReadFile(subFS, entry.Name())
			if err != nil {
				log.Println(err)
				continue
			}
			err = os.WriteFile("./template/"+entry.Name(), data, 0666)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func main() {
	// 获取版本号
	var Isversion bool
	version = "2.1.1"
	flag.BoolVar(&Isversion, "version", false, "显示版本号")
	flag.Parse()
	if Isversion {
		fmt.Println(version)
		return
	}
	config, err := initializeConfig()
	if err != nil {
		log.Fatal(err)
	}
	port := config.Port
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/version", port))
		if err != nil {
			log.Fatal("健康检查失败: ", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("健康检查失败: HTTP %d", resp.StatusCode)
		}
		return
	}
	// 初始化数据库
	models.InitSqlite()
	// 获取命令行参数
	args := os.Args
	// 如果长度小于2则没有接收到任何参数
	if len(args) < 2 {
		Run(port)
		return
	}
	// 命令行参数选择
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	var username, password string
	settingCmd.StringVar(&username, "username", "", "设置账号")
	settingCmd.StringVar(&password, "password", "", "设置密码")
	settingCmd.IntVar(&port, "port", port, "修改端口")
	switch args[1] {
	// 解析setting命令标志
	case "setting":
		settingCmd.Parse(args[2:])
		if err := settings.ResetUser(username, password); err != nil {
			log.Fatal(err)
		}
		return
	case "run":
		settingCmd.Parse(args[2:])
		if port < 1 || port > 65535 {
			log.Fatal("端口必须是 1 到 65535 之间的数字")
		}
		if err := models.SetConfig(models.Config{
			Port: port,
		}); err != nil {
			log.Fatal(err)
		}
		Run(port)
	default:
		log.Fatalf("未知命令: %s", args[1])

	}
}

func initializeConfig() (models.Config, error) {
	if err := models.ConfigInit(); err != nil {
		return models.Config{}, err
	}
	config, err := models.LoadConfig()
	if err != nil {
		return models.Config{}, err
	}
	middlewares.Secret = []byte(config.JwtSecret)
	return config, nil
}

func Run(port int) {
	// 初始化gin框架
	r := gin.Default()
	// 初始化日志配置
	utils.Loginit()
	// 初始化模板
	Templateinit()
	// 安装中间件
	r.Use(middlewares.AuthorToken) // jwt验证token
	// 设置静态资源路径
	staticFiles, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Println(err)
	}
	r.StaticFS("/static", http.FS(staticFiles))
	// 设置模板路径
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(staticFiles, "index.html")
		if err != nil {
			c.Error(err)
			return

		}
		c.Data(200, "text/html", data)
	})
	// 注册路由
	routers.User(r)
	routers.Mentus(r)
	routers.Subcription(r)
	routers.Nodes(r)
	routers.Clients(r)
	routers.Total(r)
	routers.Templates(r)
	routers.Version(r, version)
	// 启动服务
	if err := r.Run(fmt.Sprintf("0.0.0.0:%d", port)); err != nil {
		log.Fatal("服务启动失败: ", err)
	}
}
