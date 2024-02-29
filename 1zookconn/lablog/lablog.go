package lablog

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// 从环境变量中获取日志详细程度
// ref: https://blog.josejg.com/debugging-pretty/
func getVerbosity() int {
	//v := os.Getenv("VERBOSE")
	v := "./test"
	level := 0
	if v != "" {
		var err error
		level, err = strconv.Atoi(v)
		if err != nil {
			log.Fatalf("无效的日志详细程度 %v", v)
		}
	}
	return level
}

// 定义不同主题的日志类型
type LogTopic string

const (
	Client  LogTopic = "CLNT" // 客户端请求
	Commit  LogTopic = "CMIT" // 提交
	Config  LogTopic = "CONF" // 配置
	Ctrler  LogTopic = "SCTR" // 控制器
	Drop    LogTopic = "DROP" // 删除
	Error   LogTopic = "ERRO" // 错误
	Heart   LogTopic = "HART" // 心跳
	Info    LogTopic = "INFO" // 信息
	Leader  LogTopic = "LEAD" // 领导者
	Log     LogTopic = "LOG1" // 日志1
	Log2    LogTopic = "LOG2" // 日志2
	Migrate LogTopic = "MIGR" // 迁移
	Persist LogTopic = "PERS" // 持久化
	Snap    LogTopic = "SNAP" // 快照
	Server  LogTopic = "SRVR" // 服务器
	Term    LogTopic = "TERM" // 术语
	Test    LogTopic = "TEST" // 测试
	Timer   LogTopic = "TIMR" // 定时器
	Trace   LogTopic = "TRCE" // 跟踪
	Vote    LogTopic = "VOTE" // 投票
	Warn    LogTopic = "WARN" // 警告
)

var debugStart time.Time // 执行开始时间
var debugVerbosity int   // 日志详细程度

func init() {
	debugVerbosity = getVerbosity() // 获取日志详细程度
	debugStart = time.Now()         // 记录执行开始时间
	// 屏蔽默认的日期和时间输出
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

// 输出日志函数
func Debug(serverId int, topic LogTopic, format string, a ...interface{}) {
	// 根据日志详细程度判断是否需要记录此条日志
	if (debugVerbosity == 1 && topic != Info) || debugVerbosity == 2 {
		elapsedTime := time.Since(debugStart).Microseconds()          // 计算自执行开始至记录日志的时间差
		elapsedTime /= 100                                            // 将时间单位转化为毫秒
		prefix := fmt.Sprintf("%06d %v ", elapsedTime, string(topic)) // 组装日志前缀
		if serverId >= 0 {                                            // 如果是服务器日志，则带上服务器 ID
			prefix += fmt.Sprintf("S%d ", serverId)
		}
		format = prefix + format // 加上日志前缀
		log.Printf(format, a...) // 输出日志
	}
}

// 输出分片日志函数
func ShardDebug(gid int, serverId int, topic LogTopic, format string, a ...interface{}) {
	if debugVerbosity == 3 { // 日志详细程度为 3 才记录此条日志
		elapsedTime := time.Since(debugStart).Microseconds()          // 计算自执行开始至记录日志的时间差
		elapsedTime /= 100                                            // 将时间单位转化为毫秒
		prefix := fmt.Sprintf("%06d %v ", elapsedTime, string(topic)) // 组装日志前缀
		if gid >= 0 && serverId >= 0 {                                // 如果是分片日志，则带上分片 ID 和服务器 ID
			prefix += fmt.Sprintf("G%d S%d ", gid, serverId)
		}
		log.Printf(prefix+format, a...) // 输出日志
	}
}
