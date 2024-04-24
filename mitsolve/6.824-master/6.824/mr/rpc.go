package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"fmt"
	"os"
	"strconv"
)

type TaskType int

const (
	TaskTypeNone TaskType = iota
	TaskTypeMap
	TaskTypeReduce
	TaskTypeSleep
	TaskTypeExit
)

// finished task
type TaskArgs struct {
	DoneType TaskType
	Id       int
	Files    []string
}

// new task
type TaskReply struct {
	Type    TaskType
	Id      int
	Files   []string
	NReduce int
}

/*
在这里添加您的RPC定义。

为coordinator在/var/tmp中创建一个几乎唯一的UNIX域套接字名称。
不能使用当前目录，因为Athena AFS不支持UNIX域套接字。
*/
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	fmt.Printf("rpc中的coordinatorSock中的string为 %s\n", s)
	return s
}
