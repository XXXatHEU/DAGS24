package shardkv

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/lablog"
	"6.824/labrpc"
	"6.824/labutil"
	"6.824/raft"
	"6.824/shardctrler"
)

const (
	// 在每个RPC处理程序中,如果领导者在等待applyCh的applyMsg时损失了任期,
	// 处理程序将定期检查leader的currentTerm,看看任期是否发生了变化
	rpcHandlerCheckRaftTermInterval = 100
	// 快照器检查RaftStateSize的最大间隔
	// 快照器将根据RaftStateSize睡眠,
	// 如果大小变大,睡眠时间更短,检查更频繁;
	// 否则,睡眠时间更长
	snapshoterCheckInterval = 100
	// Raft状态大小达到maxraftstate的多少比例时,KVServer应该拍快照
	snapshotThresholdRatio = 0.9
	// KVServer收到并应用了多少个ApplyMsg后,应触发拍快照
	snapshoterAppliedMsgInterval = 50
	// 你的服务器需要定期轮询shardctrler以了解新的配置
	// 测试期望你的代码约每100毫秒轮询一次;
	// 更频繁也可以,但太少会造成问题
	serverRefreshConfigInterval = 100
	// 告诉客户端等待一段时间后重试
	serverWaitAndRetryInterval = 50
	// 迁移器定期检查任何待处理的迁移并为该组触发shardsMigrator
	migratorInterval = 100
)

type Op struct {
	Payload interface{}
}

// 根据Op中的Payload的具体类型,生成不同的字符串表示。
func (op Op) String() string {
	switch a := op.Payload.(type) {
	case GetArgs:
		return fmt.Sprintf("{G%s %d%s %d#%d}", a.Key, key2shard(a.Key), labutil.ToSubscript(a.ConfigNum), a.ClientId%100, a.OpId)
	case PutAppendArgs:
		if a.Op == opPut {
			return fmt.Sprintf("{P%s=%s %d%s %d#%d}", a.Key, labutil.Suffix(a.Value, 4), key2shard(a.Key), labutil.ToSubscript(a.ConfigNum), a.ClientId%100, a.OpId)
		}
		return fmt.Sprintf("{A%s+=%s %d%s %d#%d}", a.Key, labutil.Suffix(a.Value, 4), key2shard(a.Key), labutil.ToSubscript(a.ConfigNum), a.ClientId%100, a.OpId)
	case MigrateShardsArgs:
		return fmt.Sprintf("{I%s G%d-> %v}", labutil.ToSubscript(a.ConfigNum), a.Gid, a.Shards)
	default:
		return ""
	}
}

// Op equalizer
func (op *Op) Equal(another *Op) bool {
	switch a := op.Payload.(type) {
	case ClerkRequest:
		b, isClerkRequest := another.Payload.(ClerkRequest)
		return isClerkRequest && a.getClientId() == b.getClientId() && a.getOpId() == b.getOpId()
	case MigrateShardsArgs:
		b, isMigrate := another.Payload.(MigrateShardsArgs)
		return isMigrate && a.Gid == b.Gid && a.ConfigNum == b.ConfigNum && a.Shards.toShardSet().Equal(b.Shards.toShardSet())
	default:
		return false
	}
}

// channel 消息从应用程序发送到 RPC 处理程序
type requestResult struct {
	Err   Err
	Value interface{}
}

// 命令索引的应用程序到 RPC 处理程序的回复通道
type commandEntry struct {
	op      Op
	replyCh chan requestResult
}

type cache struct {
	OpId   int
	Shard  int
	Result requestResult
}

type kvTable map[string]string

// 将我的 kvTable 复制到 dst kvTable 中,dst 不能为 nil
func (src kvTable) copyInto(dst kvTable) {
	for k, v := range src {
		dst[k] = v
	}
}

type shardData struct {
	Gid       int     // 所属组
	ConfigNum int     // 该分片所在组的配置号
	Data      kvTable // 该分片的数据
}

func (s shardData) String() string {
	return fmt.Sprintf("%s @G%d", labutil.ToSubscript(s.ConfigNum), s.Gid)
}

func (s shardData) Copy() shardData {
	t := shardData{Gid: s.Gid, ConfigNum: s.ConfigNum, Data: kvTable{}}
	s.Data.copyInto(t.Data)
	return t
}

type set map[int]bool

func (s set) Equal(another set) bool {
	if len(s) != len(another) {
		return false
	}
	for e := range s {
		if !another[e] {
			return false
		}
	}
	return true
}

func (s set) toOrdered() []int {
	elems := []int{}
	for e := range s {
		elems = append(elems, e)
	}
	sort.Ints(elems)
	return elems
}

func (s set) String() (r string) {
	for _, e := range s.toOrdered() {
		r += strconv.Itoa(e)
	}
	return "[" + r + "]"
}

type shards map[int]shardData

func (s shards) toShardSet() (r set) {
	r = set{}
	for shard := range s {
		r[shard] = true
	}
	return
}

func (s shards) toGroupSet() (r set) {
	r = set{}
	for _, data := range s {
		r[data.Gid] = true
	}
	return
}

func (s shards) String() string {
	r := []string{}
	for _, shard := range s.toShardSet().toOrdered() {
		r = append(r, fmt.Sprintf("%d%s", shard, labutil.ToSubscript(s[shard].ConfigNum)))
	}
	return "[" + strings.Join(r, " ") + "]"
}

func (s shards) ByGroup() string {
	group := make(map[int]shards)
	for shard, data := range s {
		if _, ok := group[data.Gid]; !ok {
			group[data.Gid] = shards{}
		}
		group[data.Gid][shard] = data
	}

	r := []string{}
	for gid, s := range group {
		r = append(r, fmt.Sprintf("G%d%v", gid, s))
	}
	return strings.Join(r, "|")
}

// 组信息
type groupInfo struct {
	Leader           int       // 组的leader，从leader发送MigrateShards RPC开始
	Servers          []string  // 组中的服务器列表
	MigrationTrigger chan bool // 触发此组的MigrateShards RPC调用的触发器
}

type ShardKV struct {
	mu           sync.Mutex
	me           int
	rf           *raft.Raft
	sm           *shardctrler.Clerk // 碎片管理器
	dead         int32              // 由Kill()设置
	make_end     func(string) *labrpc.ClientEnd
	gid          int     // 组ID
	maxraftstate float64 // 当日志增长到一定大小时进行快照

	commandTbl           map[int]commandEntry // 由领导者维护的从命令索引到命令条目的映射，重启时初始化为空
	appliedCommandIndex  int                  // 来自applyCh的最后应用命令索引
	configFetcherTrigger chan bool            // 触发配置获取器以更新碎片配置
	migrationTrigger     chan bool            // 触发迁移器以检查任何待迁移的碎片

	// 需要在重启之间持久化的状态
	Tbl       shards          // 按碎片存储的键值表
	ClientTbl map[int64]cache // 按客户端ID存储的上一个RPC操作的结果（用于检测重复操作）

	Config     shardctrler.Config // 最新已知的碎片配置
	Cluster    map[int]*groupInfo // 组ID到组信息的映射
	MigrateOut shards             // 需要迁出的碎片
	WaitIn     shards             // 等待迁入的碎片
}

// 检查我的组是否应该为分片提供服务，加锁
func (kv *ShardKV) shouldServeShard(shard int) bool {
	return kv.Config.Shards[shard] == kv.gid
}

// 检查分片是否在我的组的WaitIn中，加锁
func (kv *ShardKV) isInWaitIn(shard int) bool {
	_, ok := kv.WaitIn[shard]
	return ok
}

// 检查此ClerkRequest是否适用于我的组，加锁
func (kv *ShardKV) checkClerkRequest(req ClerkRequest) (result requestResult, applicable bool) {
	if lastOpCache, ok := kv.ClientTbl[req.getClientId()]; ok && lastOpCache.Result.Err == OK && lastOpCache.OpId >= req.getOpId() {
		// 检测重复的成功操作，使用缓存的结果进行回复
		return lastOpCache.Result, false
	}
	if kv.Config.Num > req.getConfigNum() {
		// 请求的配置已过时，告诉客户端更新其碎片配置
		return requestResult{Err: ErrOutdatedConfig}, false
	}
	shard := req.getShard()
	if kv.Config.Num == 0 || !kv.shouldServeShard(shard) {
		// 没有获取到配置，或者不负责键的碎片
		return requestResult{Err: ErrWrongGroup}, false
	}
	if kv.isInWaitIn(shard) {
		// 键的碎片正在迁移过程中
		return requestResult{Err: ErrInMigration}, false
	}
	return result, true
}

// 通用的RPC处理程序逻辑
//
// 1. 检查是否已关闭
// 2. 检查是否为领导者
// 3. 检查请求的配置号，并在必要时触发配置获取器
// 4. 如果是Get或PutAppend RPC请求
// 4.1. 检查是否有缓存的结果
// 4.2. 检查是否可以提供所在的碎片
// 5. 启动Raft一致性
// 6. 等待来自应用程序的同意结果
// 7. 返回RPC回复结果
func (kv *ShardKV) commonHandler(op Op) (e Err, r interface{}) {
	defer func() {
		if e != OK {
			lablog.ShardDebug(kv.gid, kv.me, lablog.Error, "%s %v", string(e), op)
		}
	}()

	if kv.killed() {
		e = ErrShutdown
		return
	}

	if _, isLeader := kv.rf.GetState(); !isLeader {
		// 只有领导者才负责检查请求的配置号并启动Raft一致性
		e = ErrWrongLeader
		return
	}

	// 重要：在rf.Start之前加锁，
	// 防止Raft在kv.commandTbl为该命令索引设置replyCh之前过快完成
	kv.mu.Lock()

	switch payload := op.Payload.(type) {
	case MigrateShardsArgs:
		if kv.Config.Num < payload.ConfigNum {
			// 我的碎片配置似乎过时，告诉配置获取器更新
			kv.triggerConfigFetch()
		}
	case ClerkRequest:
		if kv.Config.Num < payload.getConfigNum() {
			// 我的碎片配置似乎过时，告诉配置获取器更新
			kv.mu.Unlock()
			kv.triggerConfigFetch()
			// 由于未来的未知配置，无法接受请求，告诉客户端稍后重试
			e = ErrUnknownConfig
			return
		}
		if result, applicable := kv.checkClerkRequest(payload); !applicable {
			kv.mu.Unlock()
			e, r = result.Err, result.Value
			return
		}
	}

	index, term, isLeader := kv.rf.Start(op)
	if term == 0 {
		kv.mu.Unlock()
		e = ErrInitElection
		return
	}
	if !isLeader {
		kv.mu.Unlock()
		e = ErrWrongLeader
		return
	}

	lablog.ShardDebug(kv.gid, kv.me, lablog.Server, "Start %v@%d", op, index)

	c := make(chan requestResult) // 用于应用程序的回复通道
	kv.commandTbl[index] = commandEntry{op: op, replyCh: c}
	kv.mu.Unlock()

CheckTermAndWaitReply:
	for !kv.killed() {
		select {
		case result, ok := <-c:
			if !ok {
				e = ErrShutdown
				return
			}
			// 从应用程序的goroutine中获取回复
			e, r = result.Err, result.Value
			return
		case <-time.After(rpcHandlerCheckRaftTermInterval * time.Millisecond):
			if t, _ := kv.rf.GetState(); term != t {
				lablog.ShardDebug(kv.gid, kv.me, lablog.Server, "Start %v@%d but NOT leader, term changed", op, index)
				e = ErrWrongLeader
				break CheckTermAndWaitReply
			}
		}
	}

	go func() { <-c }() // 避免应用程序阻塞，避免资源泄漏
	if kv.killed() {
		e = ErrShutdown
	}
	return
}

// Get RPC处理程序
func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {
	e, r := kv.commonHandler(Op{Payload: *args})
	reply.Err = e
	if e == OK {
		reply.Value = r.(string)
	}
}

// PutAppend RPC处理程序
func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	reply.Err, _ = kv.commonHandler(Op{Payload: *args})
}

/********************************* Migration **********************************/

// MigrateShards RPC处理程序
func (kv *ShardKV) MigrateShards(args *MigrateShardsArgs, reply *MigrateShardsReply) {
	reply.Err, _ = kv.commonHandler(Op{Payload: *args})
}

// MigrateShards RPC调用者，将碎片迁移到一个组
func (kv *ShardKV) migrateShards(gid int) {
	args := &MigrateShardsArgs{
		Gid: kv.gid,
	}

	for !kv.killed() {
		//检查是否是领导者
		if _, isLeader := kv.rf.GetState(); !isLeader {
			return // 非领导者，中止
		}

		// 在每次迭代中，设置需要迁移的新碎片，以防并发更新MigrateOut
		args.Shards = shards{}
		kv.mu.Lock()
		for shard, out := range kv.MigrateOut {
			if in, ok := kv.WaitIn[shard]; out.Gid == gid && (!ok || out.ConfigNum < in.ConfigNum) {
				// gid是迁出碎片的目标，
				// 并且这个碎片不在我的组的WaitIn中，
				// 或者如果在我的组的WaitIn中，这个等待的碎片在未来等待，
				// 在从我的组迁出之后，这个碎片仍然需要先迁出，然后我的组等待它迁回
				args.Shards[shard] = out.Copy()
			}
		}
		if len(args.Shards) == 0 {
			// 没有需要迁出的碎片，此次迁移已完成
			kv.mu.Unlock()
			return
		}

		// 重要：需要提供至多一次语义（重复检测），
		// 用于跨碎片移动的客户端请求
		args.ClientTbl = make(map[int64]cache)
		for clientId, cache := range kv.ClientTbl {
			if _, ok := args.Shards[cache.Shard]; ok {
				args.ClientTbl[clientId] = cache
			}
		}

		args.ConfigNum = kv.Config.Num // 更新args.ConfigNum以反映我的组的当前configNum

		servers := kv.Cluster[gid].Servers
		serverId := kv.Cluster[gid].Leader // 从目标组的最后已知leader开始
		kv.mu.Unlock()

		lablog.ShardDebug(kv.gid, kv.me, lablog.Migrate, "C%d CM->G%d %v", args.ConfigNum, gid, args.Shards)

		for i, nServer := 0, len(servers); i < nServer && !kv.killed(); {
			srv := kv.make_end(servers[serverId])
			reply := &MigrateShardsReply{}
			ok := srv.Call("ShardKV.MigrateShards", args, reply)
			if !ok || reply.Err == ErrWrongLeader || reply.Err == ErrShutdown {
				serverId = (serverId + 1) % nServer
				i++
				continue
			}

			kv.mu.Lock()
			kv.Cluster[gid].Leader = serverId // 记住目标组的这个leader
			kv.mu.Unlock()
			if reply.Err == ErrUnknownConfig {
				// 目标服务器正在尝试更新碎片配置，所以等待一段时间并在新的迭代中重试
				break
			}
			if reply.Err == ErrOutdatedConfig {
				// 我的碎片配置似乎过时，即将更新它并在新的迭代中稍后重试
				kv.triggerConfigFetch()
				break
			}
			if reply.Err == OK {
				// 迁移完成，启动Raft一致性以通知组并更新MigrateOut
				reply.Gid = gid
				reply.Shards = shards{}
				for shard, data := range args.Shards {
					reply.Shards[shard] = shardData{Gid: data.Gid, ConfigNum: data.ConfigNum, Data: nil}
				}
				_, _, isLeader := kv.rf.Start(Op{Payload: *reply})

				extra := ""
				if !isLeader {
					extra = " 但不"
				}
				lablog.ShardDebug(kv.gid, kv.me, lablog.Log2, "CM->G%d %v%s", gid, args.Shards, extra)
				return
			}
		}

		// migration not done in this turn, wait a while and retry this group
		time.Sleep(serverWaitAndRetryInterval * time.Millisecond)
	}
}

// shardsMigrator函数作为一个长期运行的goroutine，用于迁移一个组的分片。
// 当我的组首次尝试将分片迁移到目标组时创建该goroutine，
// 并在接收到触发信号时调用MigrateShards RPC到目标组。
func (kv *ShardKV) shardsMigrator(gid int, trigger <-chan bool) {
	for !kv.killed() {
		if _, ok := <-trigger; !ok {
			return
		}
		kv.migrateShards(gid)
	}
}

// 当测试器不再需要ShardKV实例时，调用Kill()方法。
// 在Kill()方法中，你不需要执行任何操作，但可能方便的做法是（例如）关闭此实例的调试输出。
func (kv *ShardKV) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
}

// 判断ShardKV实例是否已停止
func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] 包含该组中服务器的端口。
//
// me 是当前服务器在servers[]中的索引。
//
// kv服务器应该通过底层的Raft实现存储快照，
// 它应该调用persister.SaveStateAndSnapshot()来原子地保存Raft状态和快照。
//
// 当Raft的保存状态超过maxraftstate字节时，kv服务器应该进行快照，
// 以允许Raft进行日志的垃圾回收。如果maxraftstate为-1，则无需进行快照。
//
// gid 是该组的GID，用于与shardctrler进行交互。
//
// 将ctrlers[]传递给shardctrler.MakeClerk()，以便您可以发送RPC到shardctrler。
//
// make_end(servername) 将Config.Groups[gid][i]中的服务器名称转换为可以发送RPC的labrpc.ClientEnd。
// 您需要这个来向其他组发送RPC。
//
// 请查看client.go，了解如何使用ctrlers[]和make_end()发送RPC到拥有特定分片的组的示例。
//
// StartServer() 必须快速返回，因此它应该为任何长时间运行的工作启动goroutine。
//
func StartServer(
	servers []*labrpc.ClientEnd,
	me int,
	persister *raft.Persister,
	maxraftstate int,
	gid int,
	ctrlers []*labrpc.ClientEnd,
	make_end func(string) *labrpc.ClientEnd,
) *ShardKV {
	// 对希望Go的RPC库进行编组/解组的结构调用labgob.Register。
	labgob.Register(shardctrler.Config{})
	labgob.Register(Op{})
	labgob.Register(GetArgs{})
	labgob.Register(PutAppendArgs{})
	labgob.Register(MigrateShardsArgs{})
	labgob.Register(MigrateShardsReply{})
	labgob.Register(cache{})
	labgob.Register(set{})

	kv := new(ShardKV)
	kv.me = me
	kv.make_end = make_end
	kv.gid = gid
	kv.maxraftstate = float64(maxraftstate)
	applyCh := make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, applyCh)
	kv.sm = shardctrler.MakeClerk(ctrlers)

	kv.appliedCommandIndex = kv.rf.LastIncludedIndex
	kv.commandTbl = make(map[int]commandEntry)
	kv.configFetcherTrigger = make(chan bool, 1)
	kv.migrationTrigger = make(chan bool, 1)

	kv.Tbl = shards{}
	kv.ClientTbl = make(map[int64]cache)
	kv.Config = shardctrler.Config{Num: 0}
	kv.Cluster = make(map[int]*groupInfo)
	kv.MigrateOut = shards{}
	kv.WaitIn = shards{}

	// 从之前的崩溃中恢复快照
	kv.readSnapshot(persister.ReadSnapshot())

	// applier和snapshoter之间的通信，
	// 当应用了一定数量的消息时，让applier触发snapshoter进行快照
	snapshotTrigger := make(chan bool, 1)
	go kv.applier(applyCh, snapshotTrigger, kv.appliedCommandIndex)
	go kv.snapshoter(persister, snapshotTrigger)

	go kv.migrator(kv.migrationTrigger)
	kv.triggerMigration() // 如果在快照中保留了未处理的迁移，则触发任何未处理的迁移

	go kv.configFetcher(kv.configFetcherTrigger)
	kv.triggerConfigFetch() // 触发第一次配置获取

	return kv
}

// 通过持有互斥锁了解每个组的新服务器
func (kv *ShardKV) updateCluster() {
	for gid, servers := range kv.Config.Groups {
		if _, ok := kv.Cluster[gid]; !ok {
			// 创建groupInfo
			kv.Cluster[gid] = &groupInfo{Leader: 0, Servers: nil, MigrationTrigger: nil}
		}
		kv.Cluster[gid].Servers = servers
	}
}

// 将oldConfig与newConfig进行比较，回复是否可以安装newConfig
// 添加需要从我的组迁移的分片，
// 添加我组正在等待迁移数据的分片，
// 持有互斥锁
func (kv *ShardKV) updateMigrateOutAndWaitIn(oldConfig, newConfig shardctrler.Config) bool {
	if oldConfig.Num <= 0 {
		return true
	}

	addedMigrateOut, addedWaitIn := shards{}, shards{}
	for shard, oldGid := range oldConfig.Shards {
		newGid := newConfig.Shards[shard]

		out, outOk := kv.MigrateOut[shard]
		if oldGid == kv.gid && newGid != kv.gid {
			// 分片在oldConfig中的我的组中，但不在newConfig的我的组中，
			// 因此需要迁出
			switch {
			case !outOk:
				// 将该分片移动到迁移中
				addedMigrateOut[shard] = shardData{Gid: newGid, ConfigNum: newConfig.Num, Data: kv.Tbl[shard].Data}
			default:
				// 已存在相同分片的迁出，不更新分片配置
				return false
			}
		}

		in, inOk := kv.WaitIn[shard]
		if oldGid != kv.gid && newGid == kv.gid {
			// 分片不在oldConfig中的我的组中，但在当前配置的我的组中，
			// 因此需要等待
			switch {
			case !inOk:
				// 该分片尚未到达，所以我的组需要等待该分片
				addedWaitIn[shard] = shardData{Gid: kv.gid, ConfigNum: newConfig.Num, Data: nil}
			case outOk && in.ConfigNum < out.ConfigNum:
				// 该分片已经等待中，并且需要在迁入后进行迁出，
				// 为了不混淆等待迁入 -> 迁出的顺序，
				// 不更新分片配置
				return false
			case in.ConfigNum == newConfig.Num && in.Gid == kv.gid:
				// 在新配置中，该分片应该在我的组中
			default:
				lablog.ShardDebug(kv.gid, kv.me, lablog.Error, "UWI %d%v", shard, in)
				panic(shard)
			}
		}
	}

	// 检查通过，可以更新分片配置

	// 添加到MigrateOut中，从我的组的分片中删除
	for shard, out := range addedMigrateOut {
		kv.MigrateOut[shard] = out
		delete(kv.Tbl, shard)
	}
	// 添加到WaitIn中
	for shard, in := range addedWaitIn {
		kv.WaitIn[shard] = in
	}
	for shard, in := range kv.WaitIn {
		if in.Data != nil && in.ConfigNum == newConfig.Num && in.Gid == kv.gid {
			// 分片在我的组意识到分片配置更改之前就已经到达，
			// 所以当我的组知道该分片配置更改时，
			// 我的组可以愉快地接受该分片并安装到我的分片中
			kv.Tbl[shard] = in.Copy()
			delete(kv.WaitIn, shard)
		}
	}
	return true
}

// 更新我的组的分片配置（这是唯一可以更新分片配置的地方），
// 如果是领导者，则迁移分片，
// 持有互斥锁
func (kv *ShardKV) installConfig(config shardctrler.Config) {
	if config.Num <= kv.Config.Num || !kv.updateMigrateOutAndWaitIn(kv.Config, config) {
		// 过时的配置，忽略它
		// 或者无法更新MigrateOut或WaitIn，因此中止安装配置
		return
	}

	kv.Config = config
	kv.updateCluster()

	if _, isLeader := kv.rf.GetState(); !isLeader {
		// 只有领导者可以迁移任何分片
		return
	}

	lablog.ShardDebug(kv.gid, kv.me, lablog.Trace, "CN->%d MO %v,WI %v", config.Num, kv.MigrateOut.ByGroup(), kv.WaitIn.ByGroup())

	// 触发fetcher，看是否有更新的配置，以尽快更新我的分片配置
	kv.triggerConfigFetch()
	// 触发每个组的迁移
	kv.triggerMigration()
}

// 应用Get RPC请求，持有互斥锁
func (kv *ShardKV) applyGetArgs(args *GetArgs) requestResult {
	shard := key2shard(args.Key)
	return requestResult{Err: OK, Value: kv.Tbl[shard].Data[args.Key]}
}

// 在持有互斥锁的情况下应用PutAppend RPC请求
func (kv *ShardKV) applyPutAppendArgs(args *PutAppendArgs) requestResult {
	shard := key2shard(args.Key)
	if _, ok := kv.Tbl[shard]; !ok {
		kv.Tbl[shard] = shardData{Gid: kv.gid, ConfigNum: kv.Config.Num, Data: kvTable{}}
	}
	if args.Op == opPut {
		kv.Tbl[shard].Data[args.Key] = args.Value
	} else {
		kv.Tbl[shard].Data[args.Key] += args.Value
	}
	return requestResult{Err: OK, Value: nil}
}

// 接受迁移请求，安装来自该迁移的分片数据，持有互斥锁
func (kv *ShardKV) applyMigrateShardsArgs(args *MigrateShardsArgs) requestResult {
	// 回复调用方已安装的分片集合
	installed := set{}

	for shard, migration := range args.Shards {
		switch in, inOk := kv.WaitIn[shard]; {
		case migration.ConfigNum > kv.Config.Num:
			// 在我的组意识到分片配置更改之前，迁移分片已经到达，
			// 并且我的组信任迁移源组，
			// 相信这个迁移分片将在未来的配置中需要，
			// 因此接受这个迁移分片，存储到我的WaitIn中
			kv.WaitIn[shard] = migration
		case !inOk:
			// 迁移已经被接受，并且等待该分片的进程已被删除
			continue
		case migration.ConfigNum == in.ConfigNum:
			// 迁移分片与我的组的WaitIn中的相同分片匹配，
			// 并且它们属于相同的分片配置，因此我的组应该接受这个迁移分片

			switch out, outOk := kv.MigrateOut[shard]; {
			case kv.shouldServeShard(shard):
				// 我的组应该为这个分片提供服务，请将分片安装到我的组的kvTable中
				kv.Tbl[shard] = shardData{Gid: kv.gid, ConfigNum: kv.Config.Num, Data: migration.Copy().Data}
				// 从我的组的WaitIn中删除分片
				delete(kv.WaitIn, shard)
				// 告诉调用方此分片已安装到我的组
				installed[shard] = true
			case !outOk:
				// 在该分片配置下不需要迁移出，但将在以后的分片配置中迁移出
			case out.ConfigNum > in.ConfigNum:
				// 这个迁移分片需要迁移到外部，所以不需要安装到我的组的分片中，
				// 而是将这个迁移分片移动到MigrateOut中
				kv.MigrateOut[shard] = shardData{Gid: out.Gid, ConfigNum: out.ConfigNum, Data: migration.Data}
				// 从我的组的WaitIn中删除分片
				delete(kv.WaitIn, shard)
				// 告诉调用方此分片已安装到我的组
				installed[shard] = true
			default:
				lablog.ShardDebug(kv.gid, kv.me, lablog.Error, "HMI G%d-> %v, 内部默认值 %d%v", args.Gid, args.Shards, shard, in)
				panic(shard)
			}

		case migration.ConfigNum < in.ConfigNum:
			// 迁移不匹配等待，可能已经迟到
			installed[shard] = true
		default:
			lablog.ShardDebug(kv.gid, kv.me, lablog.Error, "HMI G%d-> %v, 外部默认值 %d%v", args.Gid, args.Shards, shard, in)
			panic(shard)
		}
	}

	// 为这些分片也安装客户端请求缓存
	for clientId, cache := range args.ClientTbl {
		if existingCache, ok := kv.ClientTbl[clientId]; !ok || cache.OpId > existingCache.OpId {
			kv.ClientTbl[clientId] = cache
		}
	}

	if _, isLeader := kv.rf.GetState(); isLeader {
		lablog.ShardDebug(kv.gid, kv.me, lablog.Trace, "HMI MO %v,WI %v", kv.MigrateOut.ByGroup(), kv.WaitIn.ByGroup())

		// MigrateOut已更新，触发迁移
		kv.triggerMigration()
		// WaitIn已更新，触发配置获取，可能有新的分片配置可以更新
		kv.triggerConfigFetch()
	}

	return requestResult{Err: OK, Value: installed}
}

// 迁移完成，从我的组的MigrateOut中移除分片，持有互斥锁
func (kv *ShardKV) applyMigrateShardsReply(reply *MigrateShardsReply) {
	for shard, accepted := range reply.Shards {
		if out, ok := kv.MigrateOut[shard]; ok && out.Gid == accepted.Gid && out.ConfigNum == accepted.ConfigNum {
			// 接受的迁移版本与当前迁移出的版本匹配，可以从我的组的MigrateOut中删除
			delete(kv.MigrateOut, shard)
		}
	}

	if _, isLeader := kv.rf.GetState(); isLeader {
		lablog.ShardDebug(kv.gid, kv.me, lablog.Trace, "AMO MO %v,WI %v", kv.MigrateOut.ByGroup(), kv.WaitIn.ByGroup())

		// MigrateOut已更新，触发配置获取，可能有新的分片配置可以更新
		kv.triggerConfigFetch()
		return
	}
}

// applier协程接受来自applyCh（底层raft）的applyMsg，
// 相应地修改键值表，
// 如果有的话，通过命令索引识别的通道回复修改后的结果给KVServer的RPC处理程序
// 每处理snapshoterAppliedMsgInterval条消息后，触发一个快照
func (kv *ShardKV) applier(applyCh <-chan raft.ApplyMsg, snapshotTrigger chan<- bool, lastSnapshoterTriggeredCommandIndex int) {
	defer func() {
		kv.mu.Lock()
		// 关闭所有待处理的RPC处理程序回复通道，以避免goroutine资源泄漏
		for _, ce := range kv.commandTbl {
			close(ce.replyCh)
		}
		kv.mu.Unlock()
	}()

	for m := range applyCh {
		if m.SnapshotValid {
			// 是快照，根据该快照重置kv服务器状态
			kv.mu.Lock()
			kv.appliedCommandIndex = m.SnapshotIndex
			kv.readSnapshot(m.Snapshot)
			// 清空所有待处理的回复通道，以避免goroutine资源泄漏
			for _, ce := range kv.commandTbl {
				ce.replyCh <- requestResult{Err: ErrWrongLeader}
			}
			kv.commandTbl = make(map[int]commandEntry)
			kv.mu.Unlock()
			continue
		}

		if !m.CommandValid {
			continue
		}

		if m.CommandIndex-lastSnapshoterTriggeredCommandIndex > snapshoterAppliedMsgInterval {
			// 已经应用了一定数量的消息，准备告诉快照器进行快照
			select {
			case snapshotTrigger <- true:
				lastSnapshoterTriggeredCommandIndex = m.CommandIndex // 记录上一次触发的命令索引
			default:
			}
		}

		op := m.Command.(Op)
		kv.mu.Lock()

		kv.appliedCommandIndex = m.CommandIndex

		var result requestResult
		switch payload := op.Payload.(type) {
		case ClerkRequest:
			if r, applicable := kv.checkClerkRequest(payload); !applicable {
				result = r
			} else {
				switch args := payload.(type) {
				case GetArgs:
					result = kv.applyGetArgs(&args)
				case PutAppendArgs:
					result = kv.applyPutAppendArgs(&args)
				}
				// 缓存操作结果
				kv.ClientTbl[payload.getClientId()] = cache{OpId: payload.getOpId(), Shard: payload.getShard(), Result: result}
			}
		case MigrateShardsArgs:
			result = kv.applyMigrateShardsArgs(&payload)
		case shardctrler.Config:
			kv.installConfig(payload)
			kv.mu.Unlock()
			continue
		case MigrateShardsReply:
			kv.applyMigrateShardsReply(&payload)
			kv.mu.Unlock()
			continue
		}

		ce, ok := kv.commandTbl[m.CommandIndex]
		if ok {
			delete(kv.commandTbl, m.CommandIndex) // 删除不再使用的回复通道
		}
		kv.mu.Unlock()

		// 只有领导者服务器维护commandTbl，跟随者只应用kv修改
		if ok {
			if !ce.op.Equal(&op) {
				// 您的解决方案需要处理调用Start()以进行Clerk的RPC的领导者，
				// 但在请求提交到日志之前失去了领导权。
				// 在这种情况下，您应该安排Clerk将请求重新发送给其他服务器，
				// 直到找到新的领导者为止。
				//
				// 一种方法是使服务器检测到它已经失去了领导地位，
				// 通过注意到在Start()返回的索引处出现了不同的请求来实现
				ce.replyCh <- requestResult{Err: ErrWrongLeader}
			} else {
				switch r := result.Value.(type) {
				case string: // Get
				case nil: // PutAppend
				case set: // MigrateShards
					lablog.ShardDebug(kv.gid, kv.me, lablog.Migrate, "Done %v@%d %v", op, m.CommandIndex, r)
				}
				ce.replyCh <- result
			}
		}
	}
}

// 触发迁移器
func (kv *ShardKV) triggerMigration() {
	select {
	case kv.migrationTrigger <- true:
	default:
	}
}

// migrator是一个goroutine，用于检查我的组的迁移情况，如果有需要迁出的分片，会触发相应组的shardsMigrator。
func (kv *ShardKV) migrator(trigger <-chan bool) {
	defer func() {
		kv.mu.Lock()
		// 关闭每个组的触发器，以避免goroutine资源泄漏
		for _, groupInfo := range kv.Cluster {
			if groupInfo.MigrationTrigger != nil {
				close(groupInfo.MigrationTrigger)
			}
		}
		kv.mu.Unlock()
	}()

	for !kv.killed() {
		select {
		case _, ok := <-trigger:
			if !ok {
				return
			}
		case <-time.After(migratorInterval * time.Millisecond):
			if kv.killed() {
				return
			}
		}

		if _, isLeader := kv.rf.GetState(); !isLeader {
			// 只有领导者才能迁出分片
			continue
		}

		kv.mu.Lock()
		for gid := range kv.MigrateOut.toGroupSet() {
			// 如果尚未启动，则为该组创建新的shardsMigrator
			switch group, ok := kv.Cluster[gid]; {
			case !ok:
				kv.Cluster[gid] = &groupInfo{Leader: 0, Servers: nil, MigrationTrigger: make(chan bool)}
				go kv.shardsMigrator(gid, kv.Cluster[gid].MigrationTrigger)
			case group.MigrationTrigger == nil:
				group.MigrationTrigger = make(chan bool)
				go kv.shardsMigrator(gid, kv.Cluster[gid].MigrationTrigger)
			}

			// 触发该组的分片迁移
			select {
			case kv.Cluster[gid].MigrationTrigger <- true:
			default:
			}
		}
		kv.mu.Unlock()
	}
}

// 触发configFetcher
func (kv *ShardKV) triggerConfigFetch() {
	select {
	case kv.configFetcherTrigger <- true:
	default:
	}
}

// configFetcher是一个goroutine，定期轮询shardctrler以了解新的配置信息。
func (kv *ShardKV) configFetcher(trigger <-chan bool) {
	for !kv.killed() {
		// 通过立即获取请求或定期获取请求来触发
		select {
		case _, ok := <-trigger:
			if !ok {
				return
			}
		case <-time.After(serverRefreshConfigInterval * time.Millisecond):
			if kv.killed() {
				return
			}
		}

		if _, isLeader := kv.rf.GetState(); !isLeader {
			// 只有领导者才能更新分片配置
			continue
		}

		// 重要：按顺序逐个处理重新配置
		kv.mu.Lock()
		num := kv.Config.Num + 1
		kv.mu.Unlock()

		config := kv.sm.Query(num)

		kv.mu.Lock()
		if config.Num > kv.Config.Num {
			// 获取更新的配置，启动Raft一致性来达成协议并告知所有的跟随者
			kv.rf.Start(Op{Payload: config})
		}
		kv.mu.Unlock()
	}
}

// snapshoter是一个goroutine，定期检查Raft状态大小是否接近maxraftstate阈值，如果是，则保存快照，或者如果收到applier的触发事件，也会进行快照。
func (kv *ShardKV) snapshoter(persister *raft.Persister, snapshotTrigger <-chan bool) {
	if kv.maxraftstate < 0 {
		// 不需要进行快照
		return
	}

	for !kv.killed() {
		ratio := float64(persister.RaftStateSize()) / kv.maxraftstate
		if ratio > snapshotThresholdRatio {
			// 接近阈值
			kv.mu.Lock()
			if data := kv.kvServerSnapshot(); data == nil {
				lablog.ShardDebug(kv.gid, kv.me, lablog.Error, "写入快照失败")
			} else {
				// 进行快照
				kv.rf.Snapshot(kv.appliedCommandIndex, data)
			}
			kv.mu.Unlock()

			ratio = 0.0
		}

		select {
		// 根据当前的RaftStateSize/maxraftstate比例进行休眠
		case <-time.After(time.Duration((1-ratio)*snapshoterCheckInterval) * time.Millisecond):
		// 等待触发
		case <-snapshotTrigger:
		}
	}
}

// 获取要进行快照的KVServer实例状态，同时持有互斥锁。
func (kv *ShardKV) kvServerSnapshot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(kv.Tbl) != nil ||
		e.Encode(kv.ClientTbl) != nil ||
		e.Encode(kv.Config) != nil ||
		e.Encode(kv.Cluster) != nil ||
		e.Encode(kv.MigrateOut) != nil ||
		e.Encode(kv.WaitIn) != nil {
		return nil
	}
	return w.Bytes()
}

// 恢复之前持久化的快照，同时持有互斥锁。
func (kv *ShardKV) readSnapshot(data []byte) {
	if len(data) == 0 {
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var tbl shards
	var clientTbl map[int64]cache
	var config shardctrler.Config
	var cluster map[int]*groupInfo
	var migrateOut shards
	var waitIn shards
	if d.Decode(&tbl) != nil ||
		d.Decode(&clientTbl) != nil ||
		d.Decode(&config) != nil ||
		d.Decode(&cluster) != nil ||
		d.Decode(&migrateOut) != nil ||
		d.Decode(&waitIn) != nil {
		lablog.ShardDebug(kv.gid, kv.me, lablog.Error, "读取损坏的快照")
		return
	}
	kv.Tbl = tbl
	kv.ClientTbl = clientTbl
	kv.Config = config
	kv.Cluster = cluster
	kv.MigrateOut = migrateOut
	kv.WaitIn = waitIn
}
