package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	OpaDataPath string `mapstructure:"opa_data_path"`
	AppEnv      string `mapstructure:"app_env"`
	HTTP        struct {
		ReadHeaderTimeout int `mapstructure:"read_header_timeout"`
		IdleTimeout       int `mapstructure:"idle_timeout"`
		ContextTime       int `mapstructure:"context_time"`
	} `mapstructure:"http"`
	DB struct {
		Timeout         int  `mapstructure:"timeout"`
		ReadTimeout     int  `mapstructure:"read_time_out"`
		WriteTimeout    int  `mapstructure:"write_time_out"`
		ParseTime       bool `mapstructure:"parse_time"`
		MaxOpenConns    int  `mapstructure:"max_open_conns"`
		MaxIdleConns    int  `mapstructure:"max_idle_conns"`
		ConnMaxLifetime int  `mapstructure:"conn_max_lifetime"`
		ConnMaxIdleTime int  `mapstructure:"conn_max_idle_time"`
	} `mapstructure:"db"`
	Logger struct {
		FileName   string `mapstructure:"file_name"`
		MaxSize    int    `mapstructure:"max_size"`
		MaxBackups int    `mapstructure:"max_backups"`
		MaxAge     int    `mapstructure:"max_age"`
		Compress   bool   `mapstructure:"compress"`
	} `mapstructure:"logger"`
	Security struct {
		AllowedHosts         []string          `mapstructure:"allowed_hosts"`
		SSLRedirect          bool              `mapstructure:"ssl_redirect"`
		SSLHost              string            `mapstructure:"ssl_host"`
		STSSeconds           int               `mapstructure:"sts_seconds"`
		STSIncludeSubdomains bool              `mapstructure:"sts_include_subdomains"`
		FrameDeny            bool              `mapstructure:"frame_deny"`
		ContentTypeNoSniff   bool              `mapstructure:"content_type_no_sniff"`
		IENoOpen             bool              `mapstructure:"ie_no_open"`
		ReferrerPolicy       string            `mapstructure:"referrer_policy"`
		SSLProxyHeaders      map[string]string `mapstructure:"ssl_proxy_headers"`
	} `mapstructure:"security"`
}

var Cfg Config

func LoadConfig(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config read error: %w", err)
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("config unmarshal error: %w", err)
	}

	log.Printf("[INFO] Loaded config file: %s\n", viper.ConfigFileUsed())
	return nil
}
