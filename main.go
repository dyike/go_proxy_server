package main

import (
	"flag"
	"go_proxy_server/lib"
	"go_proxy_server/server"
	"os"
	"strings"
)

func configureLog(logFile, logLevel string) error {
	level := log.DEBUG
	switch strings.ToLower(logLevel) {
	case "debug":
		level = log.DEBUG
	case "info":
		level = log.INFO
	case "warn":
		level = log.WARN
	case "error":
		level = log.ERROR
	}
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	log.Set(level, file, log.Lshortfile|log.LstdFlags)
	return nil
}

func main() {
	var logFile string
	var logLevel string
	flag.StringVar(&logFile, "l", "server.log", "日志输出路径")
	flag.StringVar(&logLevel, "ll", "debug", "日志输出等级")

	http := flag.String("http", ":8080", "proxy listen addr")
	auth := flag.String("auth", "", "basic credentitals(username:password)")
	genAuth := flag.Bool("genAuth", false, "generate credentials for auth")
	flag.Parse()

	err := configureLog(logFile, logLevel)
	if err != nil {
		log.Error("日志配置失败：%v", err)
	}

	server := server.New(*http, *auth, *genAuth)
	server.Start()
}
