package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type Coordinator struct {
	// mutex 保证对以下共享数据的互斥访问
	mutex sync.Mutex
	// mapTasks 跟踪每个 map 任务的状态
	mapTasks []MapTask
	// 剩余未完成的 map 任务数量
	mapRemain int
	// reduceTasks 跟踪每个 reduce 任务的状态
	reduceTasks []ReduceTask
	// 剩余未完成的 reduce 任务数量
	reduceRemain int
}

// MapTask 包含一个 map 任务的状态
type MapTask struct {
	id      int       // 该任务的唯一ID
	file    string    // map 任务的输入文件
	startAt time.Time // 任务开始时间
	done    bool      // 任务是否完成
}

type ReduceTask struct {
	id      int       // task id
	files   []string  // reduce task input files (M files)
	startAt time.Time // task start time (for timeout)
	done    bool      // is task finished
}

// Your code here -- RPC handlers for the worker to call.

// // GetTask 由 workers 调用获取新的 map 或 reduce任务
func (c *Coordinator) GetTask(args *TaskArgs, reply *TaskReply) error {
	// lock to protect shared data
	fmt.Printf("coordinator中的GetTask被调用！\n")
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// record result from worker for its finished task
	//判断是Reduce任务还是Map任务
	switch args.DoneType {
	case TaskTypeMap:
		if !c.mapTasks[args.Id].done {
			c.mapTasks[args.Id].done = true
			// distribute reduce files from map task result to each corresponding reduce task
			for reduceId, file := range args.Files {
				if len(file) > 0 {
					c.reduceTasks[reduceId].files = append(c.reduceTasks[reduceId].files, file)
				}
			}
			c.mapRemain--
		}
	case TaskTypeReduce:
		if !c.reduceTasks[args.Id].done {
			c.reduceTasks[args.Id].done = true
			c.reduceRemain--
		}
	}

	/**
	log.Printf("Remaining map: %d, reduce: %d\n", c.mapRemain, c.reduceRemain)
	defer log.Printf("Distribute task %v\n", reply)
	*/

	now := time.Now() //获取当前时间
	//在当前时间往前回溯10秒，从而判断那些任务超时没有完成
	//如果任务的开始时间早于这个时间并且任务还未完成则认为任务超时还未完成
	timeoutAgo := now.Add(-10 * time.Second)

	if c.mapRemain > 0 { // 还有map任务没有完成
		//遍历所有任务
		for idx := range c.mapTasks {
			t := &c.mapTasks[idx]
			// 如果任务超时而且还没有完成
			if !t.done && t.startAt.Before(timeoutAgo) {
				reply.Type = TaskTypeMap
				reply.Id = t.id
				reply.Files = []string{t.file}
				reply.NReduce = len(c.reduceTasks)

				t.startAt = now

				return nil
			}
		}
		// find no unfinished map task, ask worker to sleep for a while
		reply.Type = TaskTypeSleep
	} else if c.reduceRemain > 0 { // has remaining reduce task unfinished
		for idx := range c.reduceTasks {
			t := &c.reduceTasks[idx]
			// unfinished and timeout
			if !t.done && t.startAt.Before(timeoutAgo) {
				reply.Type = TaskTypeReduce
				reply.Id = t.id
				reply.Files = t.files

				t.startAt = now

				return nil
			}
		}
		// find no unfinished reduce task, ask worker to sleep for a while
		reply.Type = TaskTypeSleep
	} else { // both map phase and reduce phase finished, ask worker to terminate itself
		reply.Type = TaskTypeExit
	}
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	// Your code here.
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.mapRemain == 0 && c.reduceRemain == 0
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		mapTasks:     make([]MapTask, len(files)),
		reduceTasks:  make([]ReduceTask, nReduce),
		mapRemain:    len(files),
		reduceRemain: nReduce,
	}

	// Your code here.
	/**
	log.Printf(
		"Coordinator has %d map tasks and %d reduce tasks to distribute\n",
		c.mapRemain,
		c.reduceRemain,
	)
	*/

	// init map tasks
	for i, f := range files {
		c.mapTasks[i] = MapTask{id: i, file: f, done: false}
	}
	// init reduce tasks
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = ReduceTask{id: i, done: false}
	}

	c.server()
	return &c
}
