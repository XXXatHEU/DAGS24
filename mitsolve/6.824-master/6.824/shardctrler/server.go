package shardctrler

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/lablog"
	"6.824/labrpc"
	"6.824/raft"
)

const (
	// 每个 RPC 处理程序中，如果领导者在等待 applyCh 中的 applyMsg 时丢失了任期，处理程序将定期检查领导者的 currentTerm，以查看任期是否发生了变化。
	rpcHandlerCheckRaftTermInterval = 100
	// KVServer 接收和应用了多少个 ApplyMsgs 后，KVServer 应该触发进行一次快照
	snapshoterAppliedMsgInterval = 50
)

type Op struct {
	// 你的数据
	Args interface{}

	// 用于检测重复操作
	ClientId int64
	OpId     int
}

type applyResult struct {
	Err    Err
	Result Config
	OpId   int
}

type commandEntry struct {
	op      Op
	replyCh chan applyResult
}

type ShardCtrler struct {
	mu   sync.Mutex
	me   int
	rf   *raft.Raft
	dead int32 // 由 Kill() 方法设置

	commandTbl          map[int]commandEntry // 从 commandIndex 映射到 commandEntry，由 leader 维护，在重新启动时初始化为空
	appliedCommandIndex int                  // 从 applyCh 接收的最后一个已应用的 commandIndex

	// 需要在重新启动时保持
	Configs   []Config              // 配置历史记录，按配置编号索引
	ClientTbl map[int64]applyResult // 从客户端 ID 映射到最后一个 RPC 操作的结果（用于检测重复操作）
}

// Query/Join/Leave/Move RPC 处理程序的通用逻辑，与 kvraft 服务器的 RPC 处理程序非常相似（参见 ../kvraft/server.go）
func (sc *ShardCtrler) commonHandler(op Op) (e Err, r Config) {
	if sc.killed() {
		e = ErrShutdown
		return
	}

	sc.mu.Lock()
	index, term, isLeader := sc.rf.Start(op)
	if term == 0 {
		sc.mu.Unlock()
		e = ErrInitElection
		return
	}
	if !isLeader {
		sc.mu.Unlock()
		e = ErrWrongLeader
		return
	}
	c := make(chan applyResult)
	sc.commandTbl[index] = commandEntry{op: op, replyCh: c}
	sc.mu.Unlock()

CheckTermAndWaitReply:
	for !sc.killed() {
		select {
		case result, ok := <-c:
			if !ok {
				e = ErrShutdown
				return
			}
			e = result.Err
			r = result.Result
			return
		case <-time.After(rpcHandlerCheckRaftTermInterval * time.Millisecond):
			t, _ := sc.rf.GetState()
			if term != t {
				e = ErrWrongLeader
				break CheckTermAndWaitReply
			}
		}
	}

	go func() { <-c }()
	if sc.killed() {
		e = ErrShutdown
	}
	return
}

func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	reply.Err, _ = sc.commonHandler(Op{Args: *args, ClientId: args.ClientId, OpId: args.OpId})
}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {
	reply.Err, _ = sc.commonHandler(Op{Args: *args, ClientId: args.ClientId, OpId: args.OpId})
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	reply.Err, _ = sc.commonHandler(Op{Args: *args, ClientId: args.ClientId, OpId: args.OpId})
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	reply.Err, reply.Config = sc.commonHandler(Op{Args: *args, ClientId: args.ClientId, OpId: args.OpId})
}

// 当 ShardCtrler 实例不再需要时，测试器调用 Kill() 方法。你不需要在 Kill() 方法中做任何操作，但可能方便的是（例如）关闭此实例的调试输出。
func (sc *ShardCtrler) Kill() {
	atomic.StoreInt32(&sc.dead, 1)
	sc.rf.Kill()
	// 如果需要的话，在这里添加你的代码。
}

func (sc *ShardCtrler) killed() bool {
	z := atomic.LoadInt32(&sc.dead)
	return z == 1
}

// ShardCtrler 需要的 raft 实例，用于测试 shardkv
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {

	// 注册需要在RPC中序列化/反序列化的结构体
	labgob.Register(Op{})
	labgob.Register(QueryArgs{})
	labgob.Register(JoinArgs{})
	labgob.Register(LeaveArgs{})
	labgob.Register(MoveArgs{})

	// 创建ShardCtrler实例
	sc := new(ShardCtrler)

	// 设置服务器id
	sc.me = me

	// 创建应用日志的channel
	applyCh := make(chan raft.ApplyMsg)

	// 创建Raft节点
	sc.rf = raft.Make(servers, me, persister, applyCh)

	// 初始化已应用日志索引
	sc.appliedCommandIndex = sc.rf.LastIncludedIndex

	// 初始化命令表
	sc.commandTbl = make(map[int]commandEntry)

	// 初始化默认配置
	initConfig := Config{Num: 0}
	for i := range initConfig.Shards {
		initConfig.Shards[i] = 0
	}
	sc.Configs = []Config{initConfig}

	// 初始化客户端请求结果缓存
	sc.ClientTbl = make(map[int64]applyResult)

	// 从快照恢复状态
	sc.readSnapshot(persister.ReadSnapshot())

	// 创建apshot触发channel
	snapshotTrigger := make(chan bool, 1)

	// 启动applier goroutine
	go sc.applier(applyCh, snapshotTrigger, sc.appliedCommandIndex)

	// 启动snapshoter goroutine
	go sc.snapshoter(persister, snapshotTrigger)

	return sc
}

type gidShardsPair struct {
	gid    int
	shards []int
}
type gidShardsPairList []gidShardsPair

// 按 shard 负载（通过该组所持有的 shard 数量）对组进行排序
func (p gidShardsPairList) Len() int { return len(p) }
func (p gidShardsPairList) Less(i, j int) bool {
	li, lj := len(p[i].shards), len(p[j].shards)
	return li < lj || (li == lj && p[i].gid < p[j].gid)
}
func (p gidShardsPairList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// 获取配置的组负载情况
// `sortedGroupLoad` 按 shard 负载升序排序
//获取 Config 当前的组负载情况,并进行排序
// 获取配置的组负载情况
// `sortedGroupLoad` 按 shard 负载升序排序
func (cfg *Config) getGroupLoad() (groupLoad map[int][]int, sortedGroupLoad gidShardsPairList) {

	// 构建组到其持有的所有shard的映射
	groupLoad = make(map[int][]int)

	// 遍历Shards,记录每个shard对应的组id
	for shard, gid := range cfg.Shards {
		groupLoad[gid] = append(groupLoad[gid], shard)
	}

	// 对于没有分配shard的组,映射值设为空数组
	for gid := range cfg.Groups {
		if _, ok := groupLoad[gid]; !ok {
			groupLoad[gid] = []int{}
		}
	}

	// 构建排序后的 组id -> shard id数组 映射
	sortedGroupLoad = make(gidShardsPairList, len(groupLoad))

	// 按插入顺序添加元素
	i := 0
	for k, v := range groupLoad {
		sortedGroupLoad[i] = gidShardsPair{k, v}
		i++
	}

	// 按shard数量排序
	sort.Sort(sortedGroupLoad)

	return groupLoad, sortedGroupLoad
}

// 重新平衡所有组的 shards
// 目标：
// - 在完整组集合中尽可能均匀地分配 shards
// - 移动尽少的 shards
func (cfg *Config) rebalance(oldConfig *Config, joinGids []int, leaveGids []int) {
	// 获取老配置的组负载
	groupLoad, sortedGroupLoad := oldConfig.getGroupLoad()
	nOldGroups := len(sortedGroupLoad)

	// 从之前的配置中复制
	copy(cfg.Shards[:], oldConfig.Shards[:])

	switch {
	case len(cfg.Groups) == 0:
		// 没有组，将所有 shards 分配给 GID 为零的组
		for i := range cfg.Shards {
			cfg.Shards[i] = 0
		}
	case len(leaveGids) > 0:
		// 组离开，将这些组上的 shards 平均分配给其他剩余的组
		// 从低负载组开始分配
		leaveGidSet := map[int]bool{}
		for _, gid := range leaveGids {
			leaveGidSet[gid] = true
		}
		// 按顺序分配离开组的shard到其他组
		i := 0
		for _, gid := range leaveGids {
			for _, shard := range groupLoad[gid] { // 分配该组的 shards
				for leaveGidSet[sortedGroupLoad[i%nOldGroups].gid] { // 跳过离开的组
					i++
				}
				cfg.Shards[shard] = sortedGroupLoad[i%nOldGroups].gid // 分配给一个可用的组
				i++
			}
		}
	case len(joinGids) > 0:
		// 新组加入，从现有组中转移一些 shards 给这些新组
		// 从高负载组开始转移
		nNewGroups := len(cfg.Groups)
		minLoad := NShards / nNewGroups // 最小平衡负载
		nHigherLoadGroups := NShards - minLoad*nNewGroups
		joinGroupLoad := map[int]int{}
		// 计算每个新加入组的负载（shards 数量）
		for i, gid := range joinGids {
			if i >= nNewGroups-nHigherLoadGroups {
				// 一些组应该多承载一个 shard，以使整个集群负载均衡
				joinGroupLoad[gid] = minLoad + 1
			} else {
				joinGroupLoad[gid] = minLoad
			}
		}

		// 从高负载组循环转移shard给新组
		i := 0
		for _, gid := range joinGids {
			for j := 0; j < joinGroupLoad[gid]; j, i = j+1, i+1 {
				// 以循环方式获取 shards，从高负载组开始
				idx := (nOldGroups - 1 - i) % nOldGroups
				if idx < 0 {
					idx = nOldGroups + idx
				}
				sourceGroup := &sortedGroupLoad[idx]
				// 将高负载组的第一个 shard 转移到新加入组
				cfg.Shards[sourceGroup.shards[0]] = gid
				// 从原始组中删除该第一个 shard
				sourceGroup.shards = sourceGroup.shards[1:]
			}
		}
	}
}

// applier 协程从 applyCh（基于 raft）接收 applyMsg，相应地修改 shard 配置，
// 通过 commandIndex 识别的通道将配置更改回复给 Shard Controller 的 RPC 处理程序（如果有的话）
// 每经过一定数量的消息后，触发快照
func (sc *ShardCtrler) applier(applyCh <-chan raft.ApplyMsg, snapshotTrigger chan<- bool, lastSnapshoterTriggeredCommandIndex int) {
	var r Config

	for m := range applyCh {
		if m.SnapshotValid {
			// 是快照，根据此快照重置 shard controller 状态
			sc.mu.Lock()
			sc.appliedCommandIndex = m.SnapshotIndex
			sc.readSnapshot(m.Snapshot)
			// 清除所有待处理的回复通道，以避免 goroutine 资源泄漏
			for _, ce := range sc.commandTbl {
				ce.replyCh <- applyResult{Err: ErrWrongLeader}
			}
			sc.commandTbl = make(map[int]commandEntry)
			sc.mu.Unlock()
			continue
		}

		if !m.CommandValid {
			continue
		}

		if m.CommandIndex-lastSnapshoterTriggeredCommandIndex > snapshoterAppliedMsgInterval {
			// 已应用一定数量的消息，准备告诉 snapshoter 进行快照
			select {
			case snapshotTrigger <- true:
				lastSnapshoterTriggeredCommandIndex = m.CommandIndex // 记录上次触发的 commandIndex
			default:
			}
		}

		op := m.Command.(Op)
		sc.mu.Lock()

		sc.appliedCommandIndex = m.CommandIndex

		lastOpResult := sc.ClientTbl[op.ClientId]
		if lastOpResult.OpId >= op.OpId {
			// 检测到重复操作
			// 使用缓存的结果进行回复，不更新 shard 配置
			r = lastOpResult.Result
		} else {
			l := len(sc.Configs)
			switch args := op.Args.(type) {
			case QueryArgs:
				if args.Num == -1 || args.Num >= l {
					// 查询最新的配置
					args.Num = l - 1
				}
				r = sc.Configs[args.Num]
			case JoinArgs:
				lablog.ShardDebug(0, sc.me, lablog.Ctrler, "应用 Join：%v", args.Servers)
				lastConfig := sc.Configs[l-1]
				// 从先前的配置中复制
				newConfig := Config{Num: lastConfig.Num + 1, Groups: make(map[int][]string)}
				for k, v := range lastConfig.Groups {
					newConfig.Groups[k] = v
				}

				joinGids := []int{}
				for k, v := range args.Servers {
					// 添加新组
					newConfig.Groups[k] = v
					joinGids = append(joinGids, k)
				}
				// 重要：对 gid 进行排序以进行确定性的 shard 重新平衡
				sort.Ints(joinGids)
				// 进行 shards 重新平衡
				newConfig.rebalance(&lastConfig, joinGids, nil)
				// 记录新配置
				sc.Configs = append(sc.Configs, newConfig)
				r = Config{Num: -1}
			case LeaveArgs:
				lablog.ShardDebug(0, sc.me, lablog.Ctrler, "应用 Leave：%v", args.GIDs)
				lastConfig := sc.Configs[l-1]
				// 从先前的配置中复制
				newConfig := Config{Num: lastConfig.Num + 1, Groups: make(map[int][]string)}
				for k, v := range lastConfig.Groups {
					newConfig.Groups[k] = v
				}

				for _, gid := range args.GIDs {
					// 删除离开的组
					delete(newConfig.Groups, gid)
				}
				// 进行 shards 重新平衡
				newConfig.rebalance(&lastConfig, nil, args.GIDs)
				// 记录新配置
				sc.Configs = append(sc.Configs, newConfig)
				r = Config{Num: -1}
			case MoveArgs:
				lablog.ShardDebug(0, sc.me, lablog.Ctrler, "应用 Move：shard %v -> 组 %v", args.Shard, args.GID)
				lastConfig := sc.Configs[l-1]
				// 从先前的配置中复制
				newConfig := Config{Num: lastConfig.Num + 1, Groups: make(map[int][]string)}
				for k, v := range lastConfig.Groups {
					newConfig.Groups[k] = v
				}
				copy(newConfig.Shards[:], lastConfig.Shards[:])
				// 将 shard 移动到 gid
				newConfig.Shards[args.Shard] = args.GID
				// 记录新配置
				sc.Configs = append(sc.Configs, newConfig)
				r = Config{Num: -1}
			default:
				panic(args)
			}
			//r.Num 是否<0来判断是否是写入操作(Join/Leave/Move)。
			//在这些写入操作结束后,返回的 Config 都会设置 Num 为-1:
			if r.Num < 0 {
				// 记录写入操作（Join/Leave/Move）的结果
				latestConfig := sc.Configs[len(sc.Configs)-1]
				_, sortedGroupLoad := latestConfig.getGroupLoad()
				//{0:["a","b"], 1:["c","d"], 2:["e","f"]} / G0:[0,3] G1:[1,4] G2:[2]
				output := fmt.Sprintf("%d %v / ", latestConfig.Num, latestConfig.Groups)
				for _, p := range sortedGroupLoad {
					output += fmt.Sprintf("G%d：%v ", p.gid, p.shards)
				}
				lablog.ShardDebug(0, sc.me, lablog.Info, output)
			}

			// 缓存操作结果
			sc.ClientTbl[op.ClientId] = applyResult{OpId: op.OpId, Result: r}
		}

		ce, ok := sc.commandTbl[m.CommandIndex]
		if ok {
			delete(sc.commandTbl, m.CommandIndex) // 删除不再使用的回复通道
		}
		sc.mu.Unlock()

		// 只有 leader 服务器维护 shard 配置，follower 只应用配置修改
		if ok {
			if ce.op.ClientId != op.ClientId || ce.op.OpId != op.OpId {
				// leader 在请求提交到日志之前失去了领导权
				ce.replyCh <- applyResult{Err: ErrWrongLeader}
			} else {
				ce.replyCh <- applyResult{Err: OK, Result: r}
			}
		}
	}

	close(snapshotTrigger)
	// 清理所有待处理的 RPC 处理程序回复通道，避免 goroutine 资源泄漏
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for _, ce := range sc.commandTbl {
		close(ce.replyCh)
	}
}

// snapshoter 协程在从 applier 收到触发事件时进行快照
func (sc *ShardCtrler) snapshoter(persister *raft.Persister, snapshotTrigger <-chan bool) {
	for !sc.killed() {
		// 等待触发
		_, ok := <-snapshotTrigger

		if ok {
			sc.mu.Lock()
			if data := sc.shardCtrlerSnapshot(); data == nil {
				lablog.ShardDebug(0, sc.me, lablog.Error, "写入快照失败")
			} else {
				sc.rf.Snapshot(sc.appliedCommandIndex, data)
			}
			sc.mu.Unlock()
		}
	}
}

// 获取要进行快照的 shard controller 实例状态，带互斥锁
func (sc *ShardCtrler) shardCtrlerSnapshot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(sc.Configs) != nil ||
		e.Encode(sc.ClientTbl) != nil {
		return nil
	}
	return w.Bytes()
}

// 读取之前持久化的快照，带互斥锁
func (sc *ShardCtrler) readSnapshot(data []byte) {
	if len(data) == 0 {
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var configs []Config
	var clientTbl map[int64]applyResult
	if d.Decode(&configs) != nil ||
		d.Decode(&clientTbl) != nil {
		lablog.ShardDebug(0, sc.me, lablog.Error, "读取损坏的快照")
		return
	}
	sc.Configs = configs
	sc.ClientTbl = clientTbl
}
