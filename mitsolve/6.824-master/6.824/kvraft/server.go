package kvraft

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/lablog"
	"6.824/labrpc"
	"6.824/raft"
)

// RPC处理程序中，如果leader在等待applyCh的应用消息时失去了任期，
//则处理程序将定期检查leader的currentTerm，以查看是否更改了term
const rpcHandlerCheckRaftTermInterval = 100

// snapshoter检查raft状态大小的最大间隔时间，
// 如果大小增长，则休眠时间较短，并且更快地检查，
// 否则，休眠时间更长
const snapshoterCheckInterval = 100

// 在KVServer接收和应用多少ApplyMsg后，应该触发拍摄快照
const snapshoterAppliedMsgInterval = 50

// KVServer在RaftStateSize / maxraftstate的比率中应该拍摄快照
const snapshotThresholdRatio = 0.9

type Op struct {
	//您的定义
	//字段名称必须以大写字母开头，
	//否则RPC会出错。
	Type  opType
	Key   string
	Value string

	//用于检测重复操作
	ClientId int64
	OpId     int
}

func (op Op) String() string {
	switch op.Type {
	case opGet:
		return fmt.Sprintf("{G %s}", op.Key)
	case opPut:
		return fmt.Sprintf("{P %s:%s}", op.Key, op.Value)
	case opAppend:
		return fmt.Sprintf("{A %s:+%s}", op.Key, op.Value)
	default:
		return ""
	}
}

type applyResult struct {
	Err   Err
	Value string
	OpId  int
}

func (r applyResult) String() string {
	switch r.Err {
	case OK:
		if l := len(r.Value); l < 10 {
			return r.Value
		} else {
			return fmt.Sprintf("...%s", r.Value[l-10:])
		}
	default:
		return string(r.Err)
	}
}

type commandEntry struct {
	op      Op
	replyCh chan applyResult
}

type KVServer struct {
	mu   sync.Mutex // 互斥锁
	me   int        // server ID
	rf   *raft.Raft // Raft 模块实例
	dead int32      // 标志此 Server 是否被 Kill()

	maxraftstate float64 // 日志大小达到此值时需要进行快照

	// 在这里添加你的定义和注释。
	commandTbl          map[int]commandEntry // 命令索引到命令记录的映射，由 leader 维护，在重启时初始化为空。
	appliedCommandIndex int                  // 已经应用到状态机的最后一个命令索引

	// 需要持久化的数据
	Tbl       map[string]string     // 键值对表
	ClientTbl map[int64]applyResult // 客户端 ID 到最近一次 RPC 操作结果的映射（用于检测重复操作）
}

// Get方法处理客户端的“获取（get）”请求，如果server已经关闭，则返回 ErrShutdown。
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	if kv.killed() { // 如果 server 已经关闭
		reply.Err = ErrShutdown // 返回 ErrShutdown
		return
	}

	op := Op{Type: opGet, Key: args.Key, ClientId: args.ClientId, OpId: args.OpId} // 构造操作 Op
	// IMPORTANT: lock before rf.Start,
	// to avoid raft finish too quick before kv.commandTbl has set replyCh for this commandIndex
	kv.mu.Lock()                             // 加锁，以便在 rf.Start 之前设置 kv.commandTbl 的 replyCh，防止 Raft 状态机太快地完成
	index, term, isLeader := kv.rf.Start(op) // 将 op 发送到 Raft 实例中进行处理，并返回对应的 index、term 和是否是 leader
	if term == 0 {
		// OPTIMIZATION: is in startup's initial election
		// reply with error to tell client to wait for a while
		kv.mu.Unlock()
		reply.Err = ErrInitElection // 如果 term 为0，说明 Raft 正在进行初始化选举，因此返回 ErrInitElection，让客户端稍等一会儿再重试
		return
	}
	if !isLeader { // 如果当前不是 leader，直接返回 ErrWrongLeader
		kv.mu.Unlock()
		reply.Err = ErrWrongLeader
		return
	}

	lablog.Debug(kv.me, lablog.Server, "Start op %v at idx: %d, from %d of client %d", op, index, op.OpId, op.ClientId) // 记录操作日志
	c := make(chan applyResult)                                                                                         // 构建 channel c，用于接收 applier 协程的处理结果
	kv.commandTbl[index] = commandEntry{op: op, replyCh: c}                                                             // 将 index 和对应的 commandEntry 存储到 kv.commandTbl 中
	kv.mu.Unlock()
CheckTermAndWaitReply:
	for !kv.killed() {
		select {
		case result, ok := <-c:
			if !ok {
				reply.Err = ErrShutdown
				return
			}
			// get reply from applier goroutine
			lablog.Debug(kv.me, lablog.Server, "Op %v at idx: %d get %v", op, index, result) // 记录操作日志
			*reply = GetReply{Err: result.Err, Value: result.Value}                          // 将 applier 协程的处理结果赋值给 reply
			return
		case <-time.After(rpcHandlerCheckRaftTermInterval * time.Millisecond):
			t, _ := kv.rf.GetState()
			if term != t { // 如果 term 不一致，直接返回 ErrWrongLeader
				reply.Err = ErrWrongLeader
				break CheckTermAndWaitReply
			}
		}
	}

	go func() { <-c }() // 避免 applier 协程阻塞，并且避免资源泄漏
	if kv.killed() {
		reply.Err = ErrShutdown
	}
}

// PutAppend 方法处理客户端的“添加（put）”和“追加（append）”请求，如果 server 已经关闭，则返回 ErrShutdown。
func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	if kv.killed() { // 如果 server 已经关闭
		reply.Err = ErrShutdown // 返回 ErrShutdown
		return
	}

	op := Op{Type: args.Op, Key: args.Key, Value: args.Value, ClientId: args.ClientId, OpId: args.OpId} // 构造操作 Op
	kv.mu.Lock()                                                                                        // 加锁，以便在 rf.Start 之前设置 kv.commandTbl 的 replyCh，防止 Raft 状态机太快地完成
	index, term, isLeader := kv.rf.Start(op)                                                            // 将 op 发送到 Raft 实例中进行处理，并返回对应的 index、term 和是否是 leader
	if term == 0 {
		kv.mu.Unlock()
		reply.Err = ErrInitElection // 如果 term 为0，说明 Raft 正在进行初始化选举，因此返回 ErrInitElection，让客户端稍等一会儿再重试
		return
	}
	if !isLeader { // 如果当前不是 leader，直接返回 ErrWrongLeader
		kv.mu.Unlock()
		reply.Err = ErrWrongLeader
		return
	}

	lablog.Debug(kv.me, lablog.Server, "Start op %v at idx: %d, from %d of client %d", op, index, op.OpId, op.ClientId) // 记录操作日志
	c := make(chan applyResult)                                                                                         // 构建 channel c，用于接收 applier 协程的处理结果
	kv.commandTbl[index] = commandEntry{op: op, replyCh: c}                                                             // 将 index 和对应的 commandEntry 存储到 kv.commandTbl 中
	kv.mu.Unlock()
CheckTermAndWaitReply:
	for !kv.killed() {
		select {
		case result, ok := <-c:
			if !ok {
				reply.Err = ErrShutdown
				return
			}
			// get reply from applier goroutine
			lablog.Debug(kv.me, lablog.Server, "Op %v at idx: %d completed", op, index) // 记录操作日志
			reply.Err = result.Err                                                      // 将 applier 协程的处理结果赋值给 reply
			return
		case <-time.After(rpcHandlerCheckRaftTermInterval * time.Millisecond):
			t, _ := kv.rf.GetState()
			if term != t { // 如果 term 不一致，直接返回 ErrWrongLeader
				reply.Err = ErrWrongLeader
				break CheckTermAndWaitReply
			}
		}
	}

	go func() { <-c }() // 避免 applier 协程阻塞，并且避免资源泄漏
	if kv.killed() {
		reply.Err = ErrShutdown
	}
}

// Kill() 方法用于停止 KVServer 实例，当测试程序判断不再需要该实例时会调用此方法。通过 atomic.StoreInt32(&kv.dead, 1)
// 来标识该实例已经终止，killed() 方法可以检测是否已经终止
func (kv *KVServer) Kill() {
	// 使用原子操作，将 dead 设为 1
	atomic.StoreInt32(&kv.dead, 1)
	// 调用 raft.Kill() 停止 Raft 实例
	kv.rf.Kill()
	// Your code here, if desired.
}

// killed() 方法可以检测 KVServer 实例是否已经 stop
func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

// StartKVServer() 方法用于启动 KVServer，servers[] 包含协作的一组服务器的端口，me 是当前服务器在 servers[] 中的索引。
// k/v 服务器应通过底层 Raft 实现存储快照并调用 persister.SaveStateAndSnapshot() 原子地保存 Raft 状态和快照。
// 当 Raft 的保存状态超过 maxraftstate 字节时，k/v 服务器应当对其进行快照，以允许 Raft 对其日志进行垃圾回收。
// 如果 maxraftstate 为 -1，则无需进行快照。
// StartKVServer() 必须快速返回，因此它应启动任何长时间运行的工作的 goroutine。
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// 需要在这里调用 labgob.Register 注册您想让 Go 的 RPC 库进行序列化/反序列化的结构体。
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = float64(maxraftstate)
	applyCh := make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, applyCh)
	// You may need initialization code here.

	// 初始化 KV Server 的状态
	kv.appliedCommandIndex = kv.rf.LastIncludedIndex // 设置已应用日志的编号
	kv.commandTbl = make(map[int]commandEntry)       // 存储操作请求
	kv.Tbl = make(map[string]string)                 // 用于存储键值对
	kv.ClientTbl = make(map[int64]applyResult)       // 存储每个客户端的最后一个操作结果

	// 从快照初始化 Raft 状态和 KV Server 状态
	kv.readSnapshot(persister.ReadSnapshot())

	// 建立 applier 协程来处理已提交的日志条目，并更新服务器状态
	// 在已应用一定数量的日志条目之后，触发 snapshoter 来生成新的快照
	snapshotTrigger := make(chan bool, 1)
	go kv.applier(applyCh, snapshotTrigger, kv.appliedCommandIndex)
	go kv.snapshoter(persister, snapshotTrigger)

	return kv
}

// applier 协程接受 applyCh 中的 applyMsg（这些消息来自底层的 raft），并相应地修改键值对表，
// 如果有的话，通过由 commandIndex 标识的通道回复修改后的结果给 KVServer 的 RPC 处理程序。
// 每经过 snapshoterAppliedMsgInterval 条消息后，触发一个快照
func (kv *KVServer) applier(applyCh <-chan raft.ApplyMsg, snapshotTrigger chan<- bool, lastSnapshoterTriggeredCommandIndex int) {
	var r string

	for m := range applyCh {
		if m.SnapshotValid {
			// 如果是快照，则根据该快照重置 kv 服务器状态
			lablog.Debug(kv.me, lablog.Snap, "在 idx: %d 获取快照", m.SnapshotIndex)
			kv.mu.Lock()
			kv.appliedCommandIndex = m.SnapshotIndex
			kv.readSnapshot(m.Snapshot)
			// 清除所有待处理响应通道，以避免 goroutine 资源泄漏
			for _, ce := range kv.commandTbl {
				ce.replyCh <- applyResult{Err: ErrWrongLeader}
			}
			kv.commandTbl = make(map[int]commandEntry)
			kv.mu.Unlock()
			continue
		}

		if !m.CommandValid {
			continue
		}

		if m.CommandIndex-lastSnapshoterTriggeredCommandIndex > snapshoterAppliedMsgInterval {
			// 已应用某个特定数量的消息，要求进行快照
			select {
			case snapshotTrigger <- true:
				lastSnapshoterTriggeredCommandIndex = m.CommandIndex // 记录上一次触发的命令索引
			default:
			}
		}

		op := m.Command.(Op)
		kv.mu.Lock()

		kv.appliedCommandIndex = m.CommandIndex

		lastOpResult, ok := kv.ClientTbl[op.ClientId]
		if ok {
			lablog.Debug(kv.me, lablog.Server, "在 idx: %d 获取操作 %#v，上一个操作 id 为 %d", m.CommandIndex, lastOpResult.OpId)
		} else {
			lablog.Debug(kv.me, lablog.Server, "在 idx: %d 获取操作 %#v", m.CommandIndex)
		}

		if lastOpResult.OpId >= op.OpId {
			// 检测到重复操作，使用缓存的结果进行回复，不用更新 kv 表
			r = lastOpResult.Value
		} else {
			switch op.Type {
			case opGet:
				r = kv.Tbl[op.Key]
			case opPut:
				kv.Tbl[op.Key] = op.Value
				r = ""
			case opAppend:
				kv.Tbl[op.Key] = kv.Tbl[op.Key] + op.Value
				r = ""
			}

			// 缓存操作结果
			kv.ClientTbl[op.ClientId] = applyResult{Value: r, OpId: op.OpId}
		}

		ce, ok := kv.commandTbl[m.CommandIndex]
		if ok {
			delete(kv.commandTbl, m.CommandIndex) // 删除未使用的响应通道
		}
		kv.mu.Unlock()

		// 只有领导者服务器维护 commandTbl，跟随者只应用 kv 的修改
		if ok {
			lablog.Debug(kv.me, lablog.Server, "找到命令 tbl，cid 为 %v", m.CommandIndex, ce)
			if ce.op != op {
				// 如果领导者在 Clerk 的 RPC 中调用了 Start()，但在请求提交到日志之前失去了领导权，则您的解决方案需要处理该情况。
				// 在这种情况下，您应该安排 Clerk 将请求重新发送到其他服务器，直到找到新的领导者。
				//
				// 一种方法是让服务器检测是否失去领导权，
				// 通过注意到不同的请求出现在 Start() 返回的索引处
				ce.replyCh <- applyResult{Err: ErrWrongLeader}
			} else {
				ce.replyCh <- applyResult{Err: OK, Value: r}
			}
		}
	}

	// 清除所有待处理的 RPC 处理程序响应通道，以避免 goroutine 资源泄漏
	kv.mu.Lock()
	defer kv.mu.Unlock()
	for _, ce := range kv.commandTbl {
		close(ce.replyCh)
	}
}

// snapshoter go 协程定期检查raft状态大小是否接近最大raft状态阈值，
// 如果是，则保存快照，
// 或者，如果从applier接收到触发事件，则也进行快照

func (kv *KVServer) snapshoter(persister *raft.Persister, snapshotTrigger <-chan bool) {
	if kv.maxraftstate < 0 {
		// 不需要拍照
		return
	}

	for !kv.killed() {
		ratio := float64(persister.RaftStateSize()) / kv.maxraftstate
		if ratio > snapshotThresholdRatio {
			// 正在接近阈值
			kv.mu.Lock()
			if data := kv.kvServerSnapshot(); data == nil {
				lablog.Debug(kv.me, lablog.Error, "写入快照失败")
			} else {
				// 进行快照
				kv.rf.Snapshot(kv.appliedCommandIndex, data)
			}
			kv.mu.Unlock()

			ratio = 0.0
		}

		select {
		// 根据当前Raft状态大小/最大raft状态比率睡眠
		case <-time.After(time.Duration((1-ratio)*snapshoterCheckInterval) * time.Millisecond):
		// 等待触发
		case <-snapshotTrigger:
		}
	}
}

// 获取要拍摄的KVServer实例状态，带有互斥锁
func (kv *KVServer) kvServerSnapshot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(kv.Tbl) != nil ||
		e.Encode(kv.ClientTbl) != nil {
		//编码失败
		return nil
	}
	return w.Bytes()
}

// 恢复之前持久化的快照，带有互斥锁
func (kv *KVServer) readSnapshot(data []byte) {
	if len(data) == 0 { // 没有快照数据
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var tbl map[string]string
	var clientTbl map[int64]applyResult
	if d.Decode(&tbl) != nil ||
		d.Decode(&clientTbl) != nil {
		// 解码失败
		lablog.Debug(kv.me, lablog.Error, "读取损坏的快照")
		return
	}
	kv.Tbl = tbl
	kv.ClientTbl = clientTbl
}
