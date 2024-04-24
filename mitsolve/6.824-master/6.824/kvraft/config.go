package kvraft

import (
	"os"
	"testing"

	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labrpc"
	"6.824/raft"
)

// 生成长度为n的随机字符串
func randstring(n int) string {
	b := make([]byte, 2*n)
	crand.Read(b)                             // 用加密安全的随机数生成器填充字节数组b
	s := base64.URLEncoding.EncodeToString(b) // 将字节数组b转换成URL兼容base64编码的字符串s
	return s[0:n]                             // 返回前n个字符，即生成指定长度的随机字符串
}

//
func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := crand.Int(crand.Reader, max)
	x := bigx.Int64()
	return x
}

// 随机化服务器的句柄顺序
func random_handles(kvh []*labrpc.ClientEnd) []*labrpc.ClientEnd {
	sa := make([]*labrpc.ClientEnd, len(kvh))
	copy(sa, kvh)
	for i := range sa {
		j := rand.Intn(i + 1)
		sa[i], sa[j] = sa[j], sa[i]
	}
	return sa
}

type config struct {
	mu           sync.Mutex          // 互斥锁
	t            *testing.T          // 测试结构体
	net          *labrpc.Network     // 网络
	n            int                 // 服务器数量
	kvservers    []*KVServer         // KV服务器
	saved        []*raft.Persister   // 用于持久化Raft状态和快照
	endnames     [][]string          // 每个服务器发送的ClientEnd名称
	clerks       map[*Clerk][]string // 客户端与ClientEnd名称的映射
	nextClientId int                 // 下一次客户端ID
	maxraftstate int                 // Raft状态机最大存储容量
	start        time.Time           // 调用make_config（）的时间

	// begin() / end() 统计
	t0    time.Time // 记录测试test_test.go开始运行时的时间
	rpcs0 int       // 记录test_test.go开始前已完成的RPC调用次数
	ops   int32     // 记录客户端get / put / append方法调用的次数
}

// 检查测试是否超时
func (cfg config) checkTimeout() {
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}

// 清除网络连接和关闭KV服务器等必要的清理操作
func (cfg *config) cleanup() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for i := 0; i < len(cfg.kvservers); i++ {
		if cfg.kvservers[i] != nil {
			cfg.kvservers[i].Kill() // 关闭KV服务器
		}
	}
	cfg.net.Cleanup()  // 清除网络连接
	cfg.checkTimeout() // 检查是否超时
}

// 获取所有服务器上的最大日志大小
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

// 获取所有服务器上的最大快照大小
func (cfg *config) SnapshotSize() int {
	snapshotsize := 0
	for i := 0; i < cfg.n; i++ {
		n := cfg.saved[i].SnapshotSize()
		if n > snapshotsize {
			snapshotsize = n
		}
	}
	return snapshotsize
}

// 连接第i个服务器到to数组中的其他服务器
// 调用者必须持有cfg.mu锁
func (cfg *config) connectUnlocked(i int, to []int) {
	// 出站socket文件
	for j := 0; j < len(to); j++ {
		endname := cfg.endnames[i][to[j]]
		cfg.net.Enable(endname, true)
	}

	// 入站socket文件
	for j := 0; j < len(to); j++ {
		endname := cfg.endnames[to[j]][i]
		cfg.net.Enable(endname, true)
	}
}

// 连接第i个服务器到to数组中的其他服务器
func (cfg *config) connect(i int, to []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.connectUnlocked(i, to)
}

// 断开第i个服务器从from数组中的其他服务器的连接
// 调用者必须持有cfg.mu锁
func (cfg *config) disconnectUnlocked(i int, from []int) {
	// 出站socket文件
	for j := 0; j < len(from); j++ {
		if cfg.endnames[i] != nil {
			endname := cfg.endnames[i][from[j]]
			cfg.net.Enable(endname, false)
		}
	}

	// 入站socket文件
	for j := 0; j < len(from); j++ {
		if cfg.endnames[j] != nil {
			endname := cfg.endnames[from[j]][i]
			cfg.net.Enable(endname, false)
		}
	}
}

// 断开第i个服务器从from数组中的其他服务器的连接
func (cfg *config) disconnect(i int, from []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.disconnectUnlocked(i, from)
}

// 获取所有服务器编号的数组
func (cfg *config) All() []int {
	all := make([]int, cfg.n)
	for i := 0; i < cfg.n; i++ {
		all[i] = i
	}
	return all
}

// 将所有服务器互相连接。
func (cfg *config) ConnectAll() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for i := 0; i < cfg.n; i++ {
		cfg.connectUnlocked(i, cfg.All())
	}
}

// 在每个服务器名称中添加客户端的特定名称
// 将其连接到所有服务器，但现在仅启用对to[]中的服务器的连接
func (cfg *config) makeClient(to []int) *Clerk {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// 生成新的ClientEnds
	ends := make([]*labrpc.ClientEnd, cfg.n)
	endnames := make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		endnames[j] = randstring(20)
		ends[j] = cfg.net.MakeEnd(endnames[j])
		cfg.net.Connect(endnames[j], j)
	}

	ck := MakeClerk(random_handles(ends))
	cfg.clerks[ck] = endnames
	cfg.nextClientId++
	cfg.ConnectClientUnlocked(ck, to)
	return ck
}

// 这是一个用于Raft算法测试的配置文件，其中包含了连接、断开连接、关闭服务器、启动服务器等多个函数，
// 下面注释分别解释每个函数所做的事情

// 删除客户端相关信息并删除对应文件
func (cfg *config) deleteClient(ck *Clerk) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	v := cfg.clerks[ck]
	for i := 0; i < len(v); i++ {
		os.Remove(v[i])
	}
	delete(cfg.clerks, ck)
}

// 该函数会带锁调用 ConnectClientUnlocked 函数，将客户端与指定服务端连接
// caller should hold cfg.mu
func (cfg *config) ConnectClientUnlocked(ck *Clerk, to []int) {
	endnames := cfg.clerks[ck]
	for j := 0; j < len(to); j++ {
		s := endnames[to[j]]
		cfg.net.Enable(s, true)
	}
}

// 这个函数会带锁调用 ConnectClientUnlocked 函数，将客户端与指定服务端连接
func (cfg *config) ConnectClient(ck *Clerk, to []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.ConnectClientUnlocked(ck, to)
}

// 该函数会带锁调用 DisconnectClientUnlocked 函数，将客户端与指定服务端断开连接
// caller should hold cfg.mu
func (cfg *config) DisconnectClientUnlocked(ck *Clerk, from []int) {
	endnames := cfg.clerks[ck]
	for j := 0; j < len(from); j++ {
		s := endnames[from[j]]
		cfg.net.Enable(s, false)
	}
}

// 这个函数会带锁调用 DisconnectClientUnlocked 函数，将客户端与指定服务端断开连接
func (cfg *config) DisconnectClient(ck *Clerk, from []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.DisconnectClientUnlocked(ck, from)
}

// 隔离并关闭一个服务端
func (cfg *config) ShutdownServer(i int) {
	// 获取锁
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// 断开服务端与其他服务端的连接
	cfg.disconnectUnlocked(i, cfg.All())

	// 禁用客户端与该服务端的连接
	cfg.net.DeleteServer(i)

	// 新建一个持久化器
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	}

	// 将 KV Server 进程杀死
	kv := cfg.kvservers[i]
	if kv != nil {
		cfg.mu.Unlock()
		kv.Kill()
		cfg.mu.Lock()
		cfg.kvservers[i] = nil
	}
}

// 如果要重启服务器，需要先调用 ShutdownServer 函数
func (cfg *config) StartServer(i int) {
	// 获取锁
	cfg.mu.Lock()

	// 生成一个新的 ClientEnd 名称的集合
	cfg.endnames[i] = make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		cfg.endnames[i][j] = randstring(20)
	}

	// 生成一组新的 ClientEnds
	ends := make([]*labrpc.ClientEnd, cfg.n)
	for j := 0; j < cfg.n; j++ {
		ends[j] = cfg.net.MakeEnd(cfg.endnames[i][j])
		cfg.net.Connect(cfg.endnames[i][j], j)
	}

	// 新建一个持久化器
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = raft.MakePersister()
	}
	cfg.mu.Unlock()

	// 启动 KV Server 进程
	cfg.kvservers[i] = StartKVServer(ends, i, cfg.saved[i], cfg.maxraftstate)

	kvsvc := labrpc.MakeService(cfg.kvservers[i])
	rfsvc := labrpc.MakeService(cfg.kvservers[i].rf)
	srv := labrpc.MakeServer()
	srv.AddService(kvsvc)
	srv.AddService(rfsvc)
	cfg.net.AddServer(i, srv)
}

// 检查服务端是否为 Leader，如果是则返回 true 和对应的服务端编号，否则返回 false 和 0
func (cfg *config) Leader() (bool, int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	for i := 0; i < cfg.n; i++ {
		_, is_leader := cfg.kvservers[i].rf.GetState()
		if is_leader {
			return true, i
		}
	}
	return false, 0
}

// 将服务端分成 2 组，把当前 Leader 放到少数方
func (cfg *config) make_partition() ([]int, []int) {
	_, l := cfg.Leader()
	p1 := make([]int, cfg.n/2+1)
	p2 := make([]int, cfg.n/2)
	j := 0
	for i := 0; i < cfg.n; i++ {
		if i != l {
			if j < len(p1) {
				p1[j] = i
			} else {
				p2[j-len(p1)] = i
			}
			j++
		}
	}
	p2[len(p2)-1] = l
	return p1, p2
}

// 只执行一次，将 CPU 核心数设为 4
var ncpu_once sync.Once

func make_config(t *testing.T, n int, unreliable bool, maxraftstate int) *config {
	// 初始化配置结构体
	cfg := &config{}
	cfg.t = t
	cfg.net = labrpc.MakeNetwork()
	cfg.n = n
	cfg.kvservers = make([]*KVServer, cfg.n)
	cfg.saved = make([]*raft.Persister, cfg.n)
	cfg.endnames = make([][]string, cfg.n)
	cfg.clerks = make(map[*Clerk][]string)
	cfg.nextClientId = cfg.n + 1000 // 客户端 ID 从该服务端编号加上 1000 开始
	cfg.maxraftstate = maxraftstate
	cfg.start = time.Now()

	// 启动所有服务端
	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(i)
	}

	// 将所有客户端连接上
	cfg.ConnectAll()

	// 设置网络是否可靠
	cfg.net.Reliable(!unreliable)

	return cfg
}

// 获取网络 RPC 调用次数总和
func (cfg *config) rpcTotal() int {
	return cfg.net.GetTotalCount()
}

// 获取每种 RPC 的调用次数
func (cfg *config) perRpc() map[string]int {
	return cfg.net.GetPerRPC()
}

// 开始一个测试，打印测试起始信息
// e.g. cfg.begin("Test (2B): RPC counts aren't too high")
func (cfg *config) begin(description string) {
	fmt.Printf("%s ...\n", description)
	cfg.t0 = time.Now()
	cfg.rpcs0 = cfg.rpcTotal()
	atomic.StoreInt32(&cfg.ops, 0)
}

// 客户端进行一次操作
func (cfg *config) op() {
	atomic.AddInt32(&cfg.ops, 1)
}

// 结束当前测试
func (cfg *config) end() {
	cfg.checkTimeout()
	if cfg.t.Failed() == false {
		t := time.Since(cfg.t0).Seconds()  // 测试使用的真实时间
		npeers := cfg.n                    // Raft peers 数量
		nrpc := cfg.rpcTotal() - cfg.rpcs0 // RPC 调用次数
		ops := atomic.LoadInt32(&cfg.ops)  // clerk get/put/append 调用次数

		// 打印每种 RPC 的调用次数
		if len(os.Getenv("RPCCNT")) > 0 {
			for rpcName, cnt := range cfg.perRpc() {
				fmt.Printf("  %s: %d\n", rpcName, cnt)
			}
		}

		// 打印测试成功消息和性能数据
		fmt.Printf("  ... Passed --")
		fmt.Printf("  %4.1f  %d %5d %4d\n", t, npeers, nrpc, ops)
	}
}
