package main

import (
	"os"

	"github.com/sirupsen/logrus"
)

// const (
// 	// 控制输出日志信息的细节，不能控制输出的顺序和格式。
// 	// 输出的日志在每一项后会有一个冒号分隔：例如2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message
// 	Ldate         = 1 << iota     // 日期：2009/01/23
// 	Ltime                         // 时间：01:23:23
// 	Lmicroseconds                 // 微秒级别的时间：01:23:23.123123（用于增强Ltime位）
// 	Llongfile                     // 文件全路径名+行号： /a/b/c/d.go:23
// 	Lshortfile                    // 文件名+行号：d.go:23（会覆盖掉Llongfile）
// 	LUTC                          // 使用UTC时间
// 	LstdFlags     = Ldate | Ltime // 标准logger的初始值
// )

/*
	Panic：记录日志，然后panic。
Fatal：致命错误，出现错误时程序无法正常运转。输出日志后，程序退出；
Error：错误日志，需要查看原因；
Warn：警告信息，提醒程序员注意；
Info：关键操作，核心流程的日志；
Debug：一般程序中输出的调试信息；
Trace：很细粒度的信息，一般用不到；
————————————————
版权声明：本文为CSDN博主「尚墨1111」的原创文章，遵循CC 4.0 BY-SA版权协议，转载请附上原文出处链接及本声明。
原文链接：https://blog.csdn.net/qq_42647903/article/details/126158524

*/
// func initlog2() {
// 	logFile, err := os.OpenFile("./run.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
// 	if err != nil {
// 		fmt.Println("open log file failed, err:", err)
// 		return
// 	}
// 	log.SetOutput(logFile)
// 	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
// }

//定义hook格式
type MyHook struct {
}

// Levels 只定义 error 和 panic 等级的日志,其他日志等级不会触发 hook
func (h *MyHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}
func inLog(entry *logrus.Entry) error {
	f, err := os.OpenFile(logpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)

	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write([]byte(entry.Message + "\n")); err != nil {
		return err
	}
	return nil
}

// Fire 将异常日志写入到指定日志文件中 并加入数据库  注意这个函数里面不能再使用log！！！！！！！
func (h *MyHook) Fire(entry *logrus.Entry) error {

	inLog(entry) //写入日志就打开

	//几个参数分别为用户名 密码 数据库名称
	/*	dsn := "user:log@tcp(192.144.220.80:3306)/log"
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			panic(err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			fmt.Printf("数据库日志出现错误=============================	")
			panic(err)
		}

		var result sql.Result
		var err1 error
		switch entry.Level {
		case logrus.ErrorLevel:
			result, err1 = db.Exec("INSERT INTO Errorlog (msg, time) VALUES (?, ?)", entry.Message, time.Now())
		case logrus.WarnLevel:
			result, err1 = db.Exec("INSERT INTO Warnlog (msg, time) VALUES (?, ?)", entry.Message, time.Now())
		case logrus.InfoLevel:
			result, err1 = db.Exec("INSERT INTO Infolog (msg, time) VALUES (?, ?)", entry.Message, time.Now())
		case logrus.DebugLevel:
			result, err1 = db.Exec("INSERT INTO Debuglog (msg, time) VALUES (?, ?)", entry.Message, time.Now())
		case logrus.PanicLevel:
			result, err1 = db.Exec("INSERT INTO Paniclog (msg, time) VALUES (?, ?)", entry.Message, time.Now())
		case logrus.FatalLevel:
			result, err1 = db.Exec("INSERT INTO Fatallog (msg, time) VALUES (?, ?)", entry.Message, time.Now())

		}

		// 执行插入操作

		if err1 != nil {
			panic(err1)
		}

		lastInsertID, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}
		lastInsertID += 1 */
	//fmt.Printf("Inserted a new record with ID %d\n", lastInsertID)

	return nil
}

var Log = logrus.New()
var logpath string

func init() {
	// now := time.Now()
	// tt := "noMemorytime——1024——%s.log"
	// logpath = fmt.Sprintf(tt, now.Format("2006_01_02___15_04_05.000000"))
	logpath = "Log==================.log"

	// Log.SetFormatter(&logrus.TextFormatter{
	// 	DisableColors:   true,
	// 	TimestampFormat: "2006-01-02 15:04:05.000",
	// 	CallerPrettyfier: func(f *runtime.Frame) (string, string) {
	// 		filename := filepath.Base(f.File)
	// 		function := strings.TrimPrefix(filepath.Ext(f.Function), ".")
	// 		return "", filename + ":" + function + ":" + strconv.Itoa(f.Line)
	// 	},
	// })

	Log.AddHook(&MyHook{})
	Log.Out = os.Stdout
	//InfoLevel
	//DebugLevel
	//ErrorLevel
	//WarnLevel
	//FatalLevel
	//Paniclog
	Log.SetLevel(logrus.InfoLevel)

	// 为当前logrus实例设置消息输出格式为json格式.
	// 同样地,也可以单独为某个logrus实例设置日志级别和hook,这里不详细叙述.
	//log.Formatter = &logrus.JSONFormatter{}
}

func main12() {

	// 输出一条 debug 级别的日志
	//init()
	// fmt.Println("能正常运行")
	// initlog()
	// Debug(-1, Info, "This is a debug message.")

	// // 输出一条带有服务器实例信息的日志
	// gid := 1
	// serverId := 2
	// ShardDebug(gid, serverId, Info, "This is a shard debug message.")
	// log.Println("这是一条很普通的日志。")
	// v := "很普通的"
	// log.Printf("这是一条%s日志。\n", v)
	// log.Fatalln("这是一条会触发fatal的日志。")

	//日志输出位置
	// logger := log.New(os.Stdout, "<New>", log.Lshortfile|log.Ldate|log.Ltime)
	// logger.Println("这是自定义的logger记录的日志。")

	logrus.WithFields(logrus.Fields{
		"animal": "walrus",
	}).Info("1111111111111111111111111111111")

	Log.WithFields(logrus.Fields{
		"animal": "walrus",
		"size":   10,
		"添加内容":   100,
	}).Info("222222222222222222222222222222222")

	//用这种方式输出长的字段
	Log.WithFields(logrus.Fields{
		"event": "eveent",
		"topic": "topic",
		"key":   "key",
	}).Info("333333333333333333333333333333")

	//Log.Debug("444444444444444444444444444444444")
	Log.Info("55555555555555555555555555555555")
	Log.Warn("66666666666666666666666666666666666")
	Log.Error("777777777777777777777777777777777")
	Log.Fatal("Bye.")         //log之后会调用os.Exit(1)
	Log.Panic("I'm bailing.") //log之后会panic()

}
