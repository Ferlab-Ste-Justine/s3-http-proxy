package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
	"github.com/gin-gonic/gin"
)

type S3Credentials struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type ConfigS3 struct {
	Endpoint    string
	Region      string
	Bucket      string
	Tls         bool
	Credentials string
	AccessKey   string `yaml:"-"`
	SecretKey   string `yaml:"-"`
}

type ConfigServerTls struct {
	Certificate string
	Key         string
}

type ConfigServer struct {
	Port      int64
	Address   string
	BasicAuth string          `yaml:"basic_auth"`
	Tls       ConfigServerTls
	DebugMode bool            `yaml:"debug_mode"`
}

type Config struct {
	S3             ConfigS3
	Server         ConfigServer
	DownloadPrefix string       `yaml:"download_prefix"`
}

func getConfigFilePath() string {
	path := os.Getenv("S3_HTTP_PROXY_CONFIG_FILE")
	if path == "" {
		return "config.yml"
	}
	return path
}

func getS3Credentials(path string) (S3Credentials, error) {
	var a S3Credentials

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return a, errors.New(fmt.Sprintf("Error reading the credentials file: %s", err.Error()))
	}

	err = yaml.Unmarshal(b, &a)
	if err != nil {
		return a, errors.New(fmt.Sprintf("Error parsing the credentials file: %s", err.Error()))
	}

	return a, nil
}

func GetConfig() (Config, error) {
	var c Config

	b, err := ioutil.ReadFile(getConfigFilePath())
	if err != nil {
		return c, errors.New(fmt.Sprintf("Error reading the configuration file: %s", err.Error()))
	}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return c, errors.New(fmt.Sprintf("Error parsing the configuration file: %s", err.Error()))
	}

	if c.S3.Credentials != "" {
		creds, credsErr := getS3Credentials(c.S3.Credentials)
		if credsErr != nil {
			return c, credsErr
		}
		c.S3.AccessKey = creds.AccessKey
		c.S3.SecretKey = creds.SecretKey
	}

	return c, nil
}

func getAccounts(conf Config) (gin.Accounts, error) {
	var accounts gin.Accounts
	if conf.Server.BasicAuth == "" {
		return accounts, nil
	}

	b, err := ioutil.ReadFile(conf.Server.BasicAuth)
	if err != nil {
		return accounts, errors.New(fmt.Sprintf("Error reading the basic auth file: %s", err.Error()))
	}
	err = yaml.Unmarshal(b, &accounts)
	if err != nil {
		return accounts, errors.New(fmt.Sprintf("Error parsing the basic auth file: %s", err.Error()))
	}

	return accounts, nil
}