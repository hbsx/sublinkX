package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// type Config struct {
// 	ID    int
// 	Key   string
// 	Value string
// }

// Config 配置结构体
type Config struct {
	JwtSecret  string `yaml:"jwt_secret"`  // JWT密钥
	ExpireDays int    `yaml:"expire_days"` // 过期天数
	Port       int    `yaml:"port"`        // 端口号
}

var comment string = `# jwt_secret: JWT密钥
# expire_days: token 过期天数
# port: 启动端口
`

// 初始化配置
func ConfigInit() error {
	if err := os.MkdirAll("./db", 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat("./db/config.yaml"); os.IsNotExist(err) {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return fmt.Errorf("生成 JWT 密钥失败: %w", err)
		}
		// 如果不存在则创建默认配置文件
		defaultConfig := Config{
			JwtSecret:  hex.EncodeToString(secret),
			ExpireDays: 14,
			Port:       8000, // 默认端口
		}

		// 生成yaml文件
		data, err := yaml.Marshal(&defaultConfig)
		if err != nil {
			return fmt.Errorf("生成默认配置文件失败: %w", err)
		}
		data = []byte(comment + string(data)) // 添加注释
		err = os.WriteFile("./db/config.yaml", data, 0600)
		if err != nil {
			return fmt.Errorf("写入配置文件失败: %w", err)
		}
		log.Println("配置文件不存在，已创建默认配置文件")
	}
	return nil
}

// 读取配置
func ReadConfig() Config {
	cfg, err := LoadConfig()
	if err != nil {
		log.Println(err)
	}
	return cfg
}

func LoadConfig() (Config, error) {
	cfg := Config{ExpireDays: 14, Port: 8000}
	file, err := os.ReadFile("./db/config.yaml")
	if err != nil {
		return cfg, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return cfg, fmt.Errorf("配置文件格式错误: %w", err)
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("配置端口必须在 1 到 65535 之间")
	}
	if cfg.JwtSecret == "" {
		return cfg, fmt.Errorf("配置 jwt_secret 不能为空")
	}
	if cfg.ExpireDays < 1 || cfg.ExpireDays > 3650 {
		return cfg, fmt.Errorf("配置 expire_days 必须在 1 到 3650 之间")
	}
	return cfg, nil
}

// 设置配置
func SetConfig(newCfg Config) error {
	oldCfg, err := LoadConfig()
	if err != nil {
		return err
	}
	// 覆盖新的字段
	if newCfg.JwtSecret != "" {
		oldCfg.JwtSecret = newCfg.JwtSecret
	}
	if newCfg.ExpireDays != 0 {
		oldCfg.ExpireDays = newCfg.ExpireDays
	}
	if newCfg.Port != 0 {
		oldCfg.Port = newCfg.Port
	}
	// 写入文件
	data, err := yaml.Marshal(&oldCfg)
	if err != nil {
		return err
	}
	data = []byte(comment + string(data)) // 添加注释
	file, err := os.CreateTemp("./db", ".config-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), "./db/config.yaml")
}
