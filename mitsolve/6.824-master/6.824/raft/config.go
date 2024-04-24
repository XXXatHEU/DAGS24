package raft

//
// support for Raft tester.
//
// we will use the original config.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"bytes"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"6.824/labgob"
	"6.824/lablog"
	"6.824/labrpc"

	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

func randstring(n int) string {
	b := make([]byte, 2*n)
	crand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := crand.Int(crand.Reader, max)
	x := bigx.Int64()
	return x
}

type config struct {
	mu          sync.Mutex            // 互斥锁，用于保护并发访问结构体的字段
	t           *testing.T            // 测试对象，用于在测试中发出日志消息和断言
	finished    int32                 // 标志位，用于指示是否完成了一次测试
	net         *labrpc.Network       // 网络对象，用于模拟网络通信
	n           int                   // 节点（Raft 实例）的数量
	rafts       []*Raft               // 存储节点的 Raft 实例
	applyErr    []string              // 来自应用通道的错误信息
	connected   []bool                // 每个服务器是否连接在网络上
	saved       []*Persister          // 每个服务器的持久化状态
	endnames    [][]string            // 每个服务器发送给其他服务器的端口文件名
	logs        []map[int]interface{} // 每个服务器的已提交日志条目的副本
	lastApplied []int                 // 每个服务器的最后应用的日志索引
	start       time.Time             // 调用 make_config() 的时间
	// begin()/end() statistics
	t0        time.Time // test_test.go 调用 cfg.begin() 的时间
	rpcs0     int       // 开始测试时的 RPC 请求数
	cmds0     int       // 达成一致的命令数量
	bytes0    int64     // 开始测试时的字节数
	maxIndex  int       // 最大的日志索引
	maxIndex0 int       // 开始测试时的最大日志索引
}

var ncpu_once sync.Once

// make_config 函数用于创建一个 config 结构体对象，并进行初始化设置。
// 参数：
// - t：testing.T 对象，用于在测试中发出日志消息和断言。
// - n：int 类型，表示节点（Raft 实例）的数量。
// - unreliable：bool 类型，表示网络是否不可靠，即是否会有延迟和丢包。
// - snapshot：bool 类型，表示是否使用快照功能。
// 返回值：
// - *config：指向创建的 config 结构体对象的指针。
func make_config(t *testing.T, n int, unreliable bool, snapshot bool) *config {
	fmt.Println("进入make_config")
	// 打印日志信息
	lablog.Debug(-1, lablog.Config, "%d servers", n)

	// ncpu_once.Do() 代码块仅会被执行一次。
	// 该代码块用于设置一些全局参数，如随机数种子，最大CPU数等。
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		rand.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4) // 设置最大的并发执行线程数为4

	cfg := &config{} // 创建一个 config 对象
	cfg.t = t
	cfg.net = labrpc.MakeNetwork() // 创建网络对象
	cfg.n = n
	cfg.applyErr = make([]string, cfg.n)          // 创建错误信息切片，用于存储来自应用通道的错误信息
	cfg.rafts = make([]*Raft, cfg.n)              // 创建 Raft 实例切片，用于存储节点的 Raft 实例
	cfg.connected = make([]bool, cfg.n)           // 创建连接状态切片，表示每个服务器是否连接在网络上
	cfg.saved = make([]*Persister, cfg.n)         // 创建持久化状态切片，存储每个服务器的持久化状态
	cfg.endnames = make([][]string, cfg.n)        // 创建端口文件名切片，表示每个服务器发送给其他服务器的端口文件名
	cfg.logs = make([]map[int]interface{}, cfg.n) // 创建日志副本切片，用于存储每个服务器的已提交日志条目的副本
	cfg.lastApplied = make([]int, cfg.n)          // 创建最后应用的日志索引切片，用于存储每个服务器的最后应用的日志索引
	cfg.start = time.Now()                        // 设置开始时间

	cfg.setunreliable(unreliable) // 根据网络可靠性设置网络对象的参数

	cfg.net.LongDelays(true) // 设置网络对象的延迟参数为true，即引入延迟

	applier := cfg.applier
	if snapshot {
		applier = cfg.applierSnap // 根据是否使用快照功能选择应用函数
	}

	// 创建一组完整的 Raft 实例
	for i := 0; i < cfg.n; i++ {
		cfg.logs[i] = map[int]interface{}{}
		cfg.start1(i, applier)
	}

	// 连接所有服务器
	for i := 0; i < cfg.n; i++ {
		cfg.connect(i)
	}

	return cfg // 返回创建的 config 对象
}

// crash1 方法用于关闭一个 Raft 服务器并保存其持久化状态。
//
// 参数：
// - i: int 类型，表示要关闭的 Raft 服务器的索引。
func (cfg *config) crash1(i int) {
	cfg.disconnect(i)       // 断开与服务器的连接
	cfg.net.DeleteServer(i) // 禁用客户端与服务器的连接

	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// 创建新的 Persister 对象，以防止旧实例覆盖新实例的持久化状态。
	// 但是复制旧 Persister 的内容，以便始终将最后持久化的状态传递给 Make() 函数。
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	}

	rf := cfg.rafts[i]
	if rf != nil {
		cfg.mu.Unlock()
		rf.Kill() // 关闭 Raft 实例
		cfg.mu.Lock()
		cfg.rafts[i] = nil
	}

	if cfg.saved[i] != nil {
		// 读取 Raft 状态和快照，并保存到新的 Persister 对象中
		raftlog := cfg.saved[i].ReadRaftState()
		snapshot := cfg.saved[i].ReadSnapshot()
		cfg.saved[i] = &Persister{}
		cfg.saved[i].SaveStateAndSnapshot(raftlog, snapshot)
	}
}

func (cfg *config) checkLogs(i int, m ApplyMsg) (string, bool) {
	err_msg := ""
	v := m.Command
	for j := 0; j < len(cfg.logs); j++ {
		if old, oldok := cfg.logs[j][m.CommandIndex]; oldok && old != v {
			log.Printf("%v: log %v; server %v\n", i, cfg.logs[i], cfg.logs[j])
			// some server has already committed a different value for this entry!
			err_msg = fmt.Sprintf("commit index=%v server=%v %v != server=%v %v",
				m.CommandIndex, i, m.Command, j, old)
		}
	}
	_, prevok := cfg.logs[i][m.CommandIndex-1]
	cfg.logs[i][m.CommandIndex] = v
	if m.CommandIndex > cfg.maxIndex {
		cfg.maxIndex = m.CommandIndex
	}
	return err_msg, prevok
}

// applier reads message from apply ch and checks that they match the log
// contents
func (cfg *config) applier(i int, applyCh chan ApplyMsg) {
	for m := range applyCh {
		if !m.CommandValid {
			// ignore other types of ApplyMsg
		} else {
			cfg.mu.Lock()
			err_msg, prevok := cfg.checkLogs(i, m)
			cfg.mu.Unlock()
			if m.CommandIndex > 1 && !prevok {
				err_msg = fmt.Sprintf("server %v apply out of order %v", i, m.CommandIndex)
			}
			if err_msg != "" {
				log.Fatalf("apply error: %v", err_msg)
				cfg.applyErr[i] = err_msg
				// keep reading after error so that Raft doesn't block
				// holding locks...
			}
		}
	}
}

// returns "" or error string
func (cfg *config) ingestSnap(i int, snapshot []byte, index int) string {
	if snapshot == nil {
		log.Fatalf("nil snapshot")
		return "nil snapshot"
	}
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	var lastIncludedIndex int
	var xlog []interface{}
	if d.Decode(&lastIncludedIndex) != nil ||
		d.Decode(&xlog) != nil {
		log.Fatalf("snapshot decode error")
		return "snapshot Decode() error"
	}
	if index != -1 && index != lastIncludedIndex {
		err := fmt.Sprintf("server %v snapshot doesn't match m.SnapshotIndex", i)
		return err
	}
	cfg.logs[i] = map[int]interface{}{}
	for j := 0; j < len(xlog); j++ {
		cfg.logs[i][j] = xlog[j]
	}
	cfg.lastApplied[i] = lastIncludedIndex
	return ""
}

const SnapShotInterval = 10

// periodically snapshot raft state
func (cfg *config) applierSnap(i int, applyCh chan ApplyMsg) {
	cfg.mu.Lock()
	rf := cfg.rafts[i]
	cfg.mu.Unlock()
	if rf == nil {
		return // ???
	}

	for m := range applyCh {
		err_msg := ""
		if m.SnapshotValid {
			if rf.CondInstallSnapshot(m.SnapshotTerm, m.SnapshotIndex, m.Snapshot) {
				cfg.mu.Lock()
				err_msg = cfg.ingestSnap(i, m.Snapshot, m.SnapshotIndex)
				cfg.mu.Unlock()
			}
		} else if m.CommandValid {
			if m.CommandIndex != cfg.lastApplied[i]+1 {
				err_msg = fmt.Sprintf("server %v apply out of order, expected index %v, got %v", i, cfg.lastApplied[i]+1, m.CommandIndex)
			}

			if err_msg == "" {
				cfg.mu.Lock()
				var prevok bool
				err_msg, prevok = cfg.checkLogs(i, m)
				cfg.mu.Unlock()
				if m.CommandIndex > 1 && !prevok {
					err_msg = fmt.Sprintf("server %v apply out of order %v", i, m.CommandIndex)
				}
			}

			cfg.mu.Lock()
			cfg.lastApplied[i] = m.CommandIndex
			cfg.mu.Unlock()

			if (m.CommandIndex+1)%SnapShotInterval == 0 {
				w := new(bytes.Buffer)
				e := labgob.NewEncoder(w)
				e.Encode(m.CommandIndex)
				var xlog []interface{}
				for j := 0; j <= m.CommandIndex; j++ {
					xlog = append(xlog, cfg.logs[i][j])
				}
				e.Encode(xlog)
				rf.Snapshot(m.CommandIndex, w.Bytes())
			}
		} else {
			// Ignore other types of ApplyMsg.
		}
		if err_msg != "" {
			log.Fatalf("apply error: %v", err_msg)
			cfg.applyErr[i] = err_msg
			// keep reading after error so that Raft doesn't block
			// holding locks...
		}
	}
}

// start1 方法用于启动或重新启动一个 Raft 实例。
// 如果已经存在 Raft 实例，则先“杀死”该实例。
// 分配新的出站端口文件名和新的状态持久化器，以隔离之前的服务器实例，因为我们无法真正地终止之前的实例。
//
// 参数：
// - i: int 类型，表示要启动的 Raft 实例的索引。
// - applier: 函数类型，表示应用日志的函数，接收一个 int 类型参数和一个 chan ApplyMsg 类型参数。
func (cfg *config) start1(i int, applier func(int, chan ApplyMsg)) {
	cfg.crash1(i) // "杀死"现有的 Raft 实例

	// 创建新的出站 ClientEnd 名称集合，以阻止旧的崩溃实例的 ClientEnd 发送消息。
	cfg.endnames[i] = make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		cfg.endnames[i][j] = randstring(20) // 随机生成长度为20的字符串作为出站 ClientEnd 名称
	}

	// 创建新的 ClientEnd 集合
	ends := make([]*labrpc.ClientEnd, cfg.n)
	for j := 0; j < cfg.n; j++ {
		ends[j] = cfg.net.MakeEnd(cfg.endnames[i][j]) // 创建一个 ClientEnd 对象
		cfg.net.Connect(cfg.endnames[i][j], j)        // 在网络对象上建立连接
	}

	cfg.mu.Lock()

	cfg.lastApplied[i] = 0 // 最后应用的日志索引设置为0

	// 创建新的 Persister 对象，以防止旧实例覆盖新实例的持久化状态。
	// 但是复制旧 Persister 的内容，以便始终将最后持久化的状态传递给 Make() 函数。
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()

		snapshot := cfg.saved[i].ReadSnapshot()
		if len(snapshot) > 0 {
			// 模拟 KV 服务器，并立即处理快照。
			// 理想情况下，Raft 应该在 applyCh 上发送该快照...
			err := cfg.ingestSnap(i, snapshot, -1)
			if err != "" {
				cfg.t.Fatal(err)
			}
		}
	} else {
		cfg.saved[i] = MakePersister()
	}

	cfg.mu.Unlock()

	applyCh := make(chan ApplyMsg)

	rf := Make(ends, i, cfg.saved[i], applyCh) // 创建 Raft 实例

	cfg.mu.Lock()
	cfg.rafts[i] = rf // 将 Raft 实例存储到切片中
	cfg.mu.Unlock()

	go applier(i, applyCh) // 启动应用日志的 goroutine

	svc := labrpc.MakeService(rf)
	srv := labrpc.MakeServer()
	srv.AddService(svc)
	cfg.net.AddServer(i, srv) // 添加 Raft 服务器到网络对象中
}

func (cfg *config) checkTimeout() {
	// enforce a two minute real-time limit on each test
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}

func (cfg *config) checkFinished() bool {
	z := atomic.LoadInt32(&cfg.finished)
	return z != 0
}

func (cfg *config) cleanup() {
	atomic.StoreInt32(&cfg.finished, 1)
	for i := 0; i < len(cfg.rafts); i++ {
		if cfg.rafts[i] != nil {
			cfg.rafts[i].Kill()
		}
	}
	cfg.net.Cleanup()
	cfg.checkTimeout()
}

// attach server i to the net.
func (cfg *config) connect(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.connected[i] = true

	// outgoing ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.connected[j] {
			endname := cfg.endnames[i][j]
			cfg.net.Enable(endname, true)
		}
	}

	// incoming ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.connected[j] {
			endname := cfg.endnames[j][i]
			cfg.net.Enable(endname, true)
		}
	}
}

// detach server i from the net.
func (cfg *config) disconnect(i int) {
	// fmt.Printf("disconnect(%d)\n", i)

	cfg.connected[i] = false

	// outgoing ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.endnames[i] != nil {
			endname := cfg.endnames[i][j]
			cfg.net.Enable(endname, false)
		}
	}

	// incoming ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.endnames[j] != nil {
			endname := cfg.endnames[j][i]
			cfg.net.Enable(endname, false)
		}
	}
}

func (cfg *config) rpcCount(server int) int {
	return cfg.net.GetCount(server)
}

func (cfg *config) rpcTotal() int {
	return cfg.net.GetTotalCount()
}

func (cfg *config) perRpc() map[string]int {
	return cfg.net.GetPerRPC()
}

// setunreliable 方法用于设置网络对象的可靠性参数。
// 参数：
// - unrel: bool 类型，表示网络是否不可靠。
func (cfg *config) setunreliable(unrel bool) {
	cfg.net.Reliable(!unrel) // 根据传入的可靠性参数设置网络对象的可靠性
}

func (cfg *config) bytesTotal() int64 {
	return cfg.net.GetTotalBytes()
}

func (cfg *config) setlongreordering(longrel bool) {
	cfg.net.LongReordering(longrel)
}

// checkOneLeader 方法用于检查已连接的服务器中是否有一个服务器认为自己是 leader，并且其他服务器没有其他认为是 leader 的情况。
// 为了确保选举进行，会尝试几次。
//
// 返回值：
// - int 类型，表示被选为 leader 的服务器的索引。
func (cfg *config) checkOneLeader() int {
	for iters := 0; iters < 10; iters++ {
		ms := 450 + (rand.Int63() % 100)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		leaders := make(map[int][]int)
		for i := 0; i < cfg.n; i++ {
			if cfg.connected[i] {
				if term, leader := cfg.rafts[i].GetState(); leader {
					leaders[term] = append(leaders[term], i)
				}
			}
		}

		lastTermWithLeader := -1
		for term, leaders := range leaders {
			if len(leaders) > 1 {
				cfg.t.Fatalf("term %d has %d (>1) leaders", term, len(leaders))
			}
			if term > lastTermWithLeader {
				lastTermWithLeader = term
			}
		}

		if len(leaders) != 0 {
			return leaders[lastTermWithLeader][0] // 返回最后一个 term 中被选为 leader 的服务器的索引
		}
	}
	cfg.t.Fatalf("expected one leader, got none")
	return -1
}

// check that everyone agrees on the term.
func (cfg *config) checkTerms() int {
	term := -1
	for i := 0; i < cfg.n; i++ {
		if cfg.connected[i] {
			xterm, _ := cfg.rafts[i].GetState()
			if term == -1 {
				term = xterm
			} else if term != xterm {
				cfg.t.Fatalf("servers disagree on term")
			}
		}
	}
	return term
}

//
// check that none of the connected servers
// thinks it is the leader.
//
func (cfg *config) checkNoLeader() {
	for i := 0; i < cfg.n; i++ {
		if cfg.connected[i] {
			_, is_leader := cfg.rafts[i].GetState()
			if is_leader {
				cfg.t.Fatalf("expected no leader among connected servers, but %v claims to be leader", i)
			}
		}
	}
}

// how many servers think a log entry is committed?
func (cfg *config) nCommitted(index int) (int, interface{}) {
	count := 0
	var cmd interface{} = nil
	for i := 0; i < len(cfg.rafts); i++ {
		if cfg.applyErr[i] != "" {
			cfg.t.Fatal(cfg.applyErr[i])
		}

		cfg.mu.Lock()
		cmd1, ok := cfg.logs[i][index]
		cfg.mu.Unlock()

		if ok {
			if count > 0 && cmd != cmd1 {
				cfg.t.Fatalf("committed values do not match: index %v, %v, %v",
					index, cmd, cmd1)
			}
			count += 1
			cmd = cmd1
		}
	}
	return count, cmd
}

// wait for at least n servers to commit.
// but don't wait forever.
func (cfg *config) wait(index int, n int, startTerm int) interface{} {
	to := 10 * time.Millisecond
	for iters := 0; iters < 30; iters++ {
		nd, _ := cfg.nCommitted(index)
		if nd >= n {
			break
		}
		time.Sleep(to)
		if to < time.Second {
			to *= 2
		}
		if startTerm > -1 {
			for _, r := range cfg.rafts {
				if t, _ := r.GetState(); t > startTerm {
					// someone has moved on
					// can no longer guarantee that we'll "win"
					return -1
				}
			}
		}
	}
	nd, cmd := cfg.nCommitted(index)
	if nd < n {
		cfg.t.Fatalf("only %d decided for index %d; wanted %d",
			nd, index, n)
	}
	return cmd
}

// do a complete agreement.
// it might choose the wrong leader initially,
// and have to re-submit after giving up.
// entirely gives up after about 10 seconds.
// indirectly checks that the servers agree on the
// same value, since nCommitted() checks this,
// as do the threads that read from applyCh.
// returns index.
// if retry==true, may submit the command multiple
// times, in case a leader fails just after Start().
// if retry==false, calls Start() only once, in order
// to simplify the early Lab 2B tests.
func (cfg *config) one(cmd interface{}, expectedServers int, retry bool) int {
	t0 := time.Now()
	starts := 0
	for time.Since(t0).Seconds() < 10 && !cfg.checkFinished() {
		// try all the servers, maybe one is the leader.
		index := -1
		for si := 0; si < cfg.n; si++ {
			starts = (starts + 1) % cfg.n
			var rf *Raft
			cfg.mu.Lock()
			if cfg.connected[starts] {
				rf = cfg.rafts[starts]
			}
			cfg.mu.Unlock()
			if rf != nil {
				index1, _, ok := rf.Start(cmd)
				if ok {
					index = index1
					break
				}
			}
		}

		if index != -1 {
			// somebody claimed to be the leader and to have
			// submitted our command; wait a while for agreement.
			t1 := time.Now()
			for time.Since(t1).Seconds() < 2 {
				nd, cmd1 := cfg.nCommitted(index)
				if nd > 0 && nd >= expectedServers {
					// committed
					if cmd1 == cmd {
						// and it was the command we submitted.
						return index
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !retry {
				cfg.t.Fatalf("one(%v) failed to reach agreement, no retry", cmd)
			}
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if !cfg.checkFinished() {
		cfg.t.Fatalf("one(%v) failed to reach agreement, not finished", cmd)
	}
	return -1
}

// start a Test.
// print the Test message.
// e.g. cfg.begin("Test (2B): RPC counts aren't too high")
func (cfg *config) begin(description string) {
	fmt.Printf("%s ...\n", description)
	cfg.t0 = time.Now()
	cfg.rpcs0 = cfg.rpcTotal()
	cfg.bytes0 = cfg.bytesTotal()
	cfg.cmds0 = 0
	cfg.maxIndex0 = cfg.maxIndex
}

// end a Test -- the fact that we got here means there
// was no failure.
// print the Passed message,
// and some performance numbers.
func (cfg *config) end() {
	cfg.checkTimeout()
	if !cfg.t.Failed() {
		cfg.mu.Lock()
		t := time.Since(cfg.t0).Seconds()       // real time
		npeers := cfg.n                         // number of Raft peers
		nrpc := cfg.rpcTotal() - cfg.rpcs0      // number of RPC sends
		nbytes := cfg.bytesTotal() - cfg.bytes0 // number of bytes
		ncmds := cfg.maxIndex - cfg.maxIndex0   // number of Raft agreements reported
		cfg.mu.Unlock()

		if len(os.Getenv("RPCCNT")) > 0 {
			for rpcName, cnt := range cfg.perRpc() {
				fmt.Printf("  %s: %d\n", rpcName, cnt)
			}
		}

		fmt.Printf("  ... Passed --")
		fmt.Printf("  %4.1f  %d %4d %7d %4d\n", t, npeers, nrpc, nbytes, ncmds)
	}
}

// Maximum log size across all servers
func (cfg *config) LogSize() int {
	logsize := 0
	for i := 0; i < cfg.n; i++ {
		n := cfg.saved[i].RaftStateSize()
		if n > logsize {
			logsize = n
		}
	}
	return logsize
}
