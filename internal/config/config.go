package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	LLM       LLMConfig       `yaml:"llm"`
	DB        DatabaseConfig  `yaml:"db"`
	OSS       OSSConfig       `yaml:"oss"`
	Admin     AdminConfig     `yaml:"admin"`
	Dashscope DashscopeConfig `yaml:"dashscope"`
}

type ServerConfig struct {
	Addr  string `yaml:"addr"`
	Debug bool   `yaml:"debug"`
}

type LLMConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	AK      string `yaml:"ak"`
	SK      string `yaml:"sk"`
	Region  string `yaml:"region"`
	Model   string `yaml:"model"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	Params   string `yaml:"params"`
}

type OSSConfig struct {
	Endpoint             string `yaml:"endpoint"`
	Region               string `yaml:"region"`
	Bucket               string `yaml:"bucket"`
	AccessKeyID          string `yaml:"access_key_id"`
	AccessKeySecret      string `yaml:"access_key_secret"`
	SecurityToken        string `yaml:"security_token"`
	Prefix               string `yaml:"prefix"`
	TempURLExpireSeconds int    `yaml:"temp_url_expire_seconds"`
}

func (o OSSConfig) Enabled() bool {
	return o.Bucket != "" && o.AccessKeyID != "" && o.AccessKeySecret != ""
}

type AdminConfig struct {
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Nickname   string `yaml:"nickname"`
	TotalQuota int64  `yaml:"total_quota"`
}

type DashscopeConfig struct {
	APIKey string                 `yaml:"api_key"`
	STT    DashscopeServiceConfig `yaml:"stt"`
	TTS    DashscopeServiceConfig `yaml:"tts"`
}

type DashscopeServiceConfig struct {
	Model    string `yaml:"model"`
	Endpoint string `yaml:"endpoint"`
	Voice    string `yaml:"voice"`
}

func (d DatabaseConfig) DSN() string {
	host := d.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := d.Port
	if port == 0 {
		port = 3306
	}
	params := d.Params
	if params == "" {
		params = "charset=utf8mb4&parseTime=true&loc=Local"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", d.User, d.Password, host, port, d.Name, params)
}

// Load 从给定路径读取 YAML 配置。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}

	return &cfg, nil
}
