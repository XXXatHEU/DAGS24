package main

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	// 设置日志格式为json格式
	log.SetFormatter(&logrus.JSONFormatter{})

	// 设置将日志输出到标准输出（默认的输出为stderr,标准错误）
	// 日志消息输出可以是任意的io.writer类型
	log.SetOutput(os.Stdout)

	// 设置日志级别为warn以上
	log.SetLevel(logrus.InfoLevel)
}

// logrus提供了New()函数来创建一个logrus的实例.
// 项目中,可以创建任意数量的logrus实例.

func main() {
	// 为当前logrus实例设置消息的输出,同样地,
	// 可以设置logrus实例的输出到任意io.writer
	log.Out = os.Stdout

	// 为当前logrus实例设置消息输出格式为json格式.
	// 同样地,也可以单独为某个logrus实例设置日志级别和hook,这里不详细叙述.
	log.Formatter = &logrus.JSONFormatter{}

	log.WithFields(logrus.Fields{
		"animal": "walrus",
		"size":   10,
	}).Info("A group of walrus emerges from the ocean")
}
