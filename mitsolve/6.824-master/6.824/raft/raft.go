package raft

/*
rf = Make(...)
// 创建一个新的Raft服务器

rf.Start(command interface{}) (index, term, isleader)
// 启动对新日志条目的协议
// 参数 command 为需要追加到日志中的命令
// 返回值 index 为该条目在日志中的索引，term 为该条目的任期号，isleader 表示当前服务器是否是领导者

rf.GetState() (term, isLeader)
// 请求 Raft 服务器的当前任期号和是否认为自己是领导者的状态信息
// 返回值 term 为当前任期号，isLeader 表示当前服务器是否是领导者

ApplyMsg
// 当每个新条目被提交到日志中时，每个 Raft peer都应该向服务（或测试器)发送一个 ApplyMsg，以便在同一台服务器上处理

*/

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/lablog"
	"6.824/labrpc"
	"6.824/labutil"
)

const (
	// 用于表示投票给谁的特殊值，-1 表示没有给任何服务器投票
	voteForNull = -1

	// 选举超时时间范围，单位为毫秒，即节点等待其它节点响应的时间范围
	electionTimeoutMax = 1200
	electionTimeoutMin = 800

	// 心跳间隔，单位为毫秒（每秒发送10个心跳RPC)
	// 心跳间隔应比选举超时时间小一个数量级
	heartbeatInterval = 100

	// 当进行快照时，领导者将会保留少量未被压缩的日志，以应对稍微滞后的跟随者节点。该参数表示保留的日志数量。
	leaderKeepLogAmount = 20
)

type serverState string

const (
	Leader    serverState = "L"
	Candidate serverState = "C"
	Follower  serverState = "F"
)

// 设置正常的选举超时时间，带有随机性
func nextElectionAlarm() time.Time {
	return time.Now().Add(time.Duration(labutil.RandRange(electionTimeoutMin, electionTimeoutMax)) * time.Millisecond)
}

// 在启动时设置快速的选举超时时间，带有随机性
func initElectionAlarm() time.Time {
	return time.Now().Add(time.Duration(labutil.RandRange(0, electionTimeoutMax-electionTimeoutMin)) * time.Millisecond)
}

// AppendEntries RPC 调用的实际意图名称
func intentOfAppendEntriesRPC(args *AppendEntriesArgs) string {
	if len(args.Entries) == 0 {
		return "HB"
	}
	return "AE"
}

func dTopicOfAppendEntriesRPC(args *AppendEntriesArgs, defaultTopic lablog.LogTopic) lablog.LogTopic {
	if len(args.Entries) == 0 {
		return lablog.Heart
	}
	return defaultTopic
}

/*
随着每个 Raft peer意识到连续的日志条目已被提交，该peer应通过传递给 Make() 的 applyCh，
向在同一服务器上的服务（或测试器)发送一个 ApplyMsg。
将 CommandValid 设置为 true 表示 ApplyMsg 包含一个新提交的日志条目。

在第2D部分中，您需要在 applyCh 上发送其他类型的消息（例如快照)，但对于这些其他用途，
将 CommandValid 设置为 false。
*/
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// Raft日志记录条目
type LogEntry struct {
	Index   int         // 日志索引
	Term    int         // 接收该条目的领导者的任期号
	Command interface{} // 由该日志条目携带的客户端命令
}

// 字符串化器
func (e LogEntry) String() string {
	commandStr := fmt.Sprintf("%v", e.Command)
	if len(commandStr) > 15 {
		commandStr = commandStr[:15] // 如果命令字符串长度超过15，就截断前15个字符
	}
	return fmt.Sprintf("{I:%d T:%d C:%s}", e.Index, e.Term, commandStr)
}

// 快照命令
// 在快照函数中接收，通过通道发送给快照处理器
type snapshotCmd struct {
	index    int
	snapshot []byte
}

// 单个Raft节点的Go对象。
type Raft struct {
	mu        sync.Mutex          // 用于保护对该节点状态的共享访问的锁
	peers     []*labrpc.ClientEnd // 所有节点的RPC端点，包括本节点
	persister *Persister          // 用于保存此节点持久状态的对象，并在最初包含最新保存的状态（如果有)
	me        int                 // 该节点在peers[]数组中的索引
	dead      int32               // 由Kill()设置

	// 如果为真，则告诉提交器前进提交索引并将msg应用于客户端
	// 如果为假，则告诉提交器将接收到的快照发送回服务
	commitTrigger   chan bool
	snapshotCh      chan snapshotCmd // 告诉快照处理器拍摄快照
	snapshotTrigger chan bool        // 告诉快照处理器保存延迟的快照

	// 你的数据 (2A, 2B, 2C)
	// 参考论文Figure 2中Raft服务器必须维护的状态描述。

	// 所有节点的持久状态，更新后会存储在稳定存储器中，然后才会响应RPC。
	CurrentTerm       int        // 服务器看到的最新任期号，单调递增
	VotedFor          int        // 当前任期内收到选票的候选人ID（如果没有则为null)
	Log               []LogEntry // 日志条目  这个很重要
	LastIncludedIndex int        // 日志中紧随在快照之前的日志条目的索引号，单调递增
	LastIncludedTerm  int        // 日志中紧随在快照之前的日志条目的任期号

	// 所有节点的易失性状态
	commitIndex   int         // 已知已提交的最高日志条目的索引号，单调递增
	lastApplied   int         // 应用于状态机的最高日志条目的索引号，单调递增
	state         serverState // 当前服务器状态（每次启动时，服务器都作为Follower启动)
	electionAlarm time.Time   // 选举超时闹钟，如果时间超过闹钟，服务器启动新的选举

	// 领导者的易失性状态，在选举后重新初始化
	nextIndex         []int      // 每个节点，要发送到该节点的下一个日志条目的索引号
	matchIndex        []int      // 每个节点，已知已复制到该节点的最高日志条目的索引号
	appendEntriesCh   []chan int // 每个跟随者，触发AppendEntries RPC调用的通道（如果是重试，则发送serialNo，否则为0)
	installSnapshotCh []chan int // 每个跟随者，触发InstallSnapshot RPC调用的通道（如果是重试，则发送serialNo，否则为0)
}

// 服务或测试人员想要创建Raft服务器。所有Raft服务器（包括本机)的端口都在peers[]中。
// 该服务器的端口是peers[me]。所有服务器的peers[]数组具有相同的顺序。
// persister是为此服务器保存其持久状态的地方，并且最初包含最新保存的状态（如果有)。
// applyCh是测试人员或服务期望Raft发送ApplyMsg消息的通道。
// Make()必须快速返回，因此应为任何长时间运行的工作启动goroutine。

// 创建一个新的Raft服务器
func Make(
	peers []*labrpc.ClientEnd,
	me int,
	persister *Persister,
	applyCh chan<- ApplyMsg, //chan<- 表示这是一个单向的通道类型，只能用于发送 ApplyMsg 类型的值，而不能进行接收操作。
) *Raft {
	// 你的初始化代码 (2A, 2B, 2C)

	//初始化一个raft结构体变量rf
	rf := &Raft{
		peers:     peers,
		persister: persister,
		me:        me,
		dead:      0,

		// 重要提示：缓冲通道
		// （未缓冲的通道可能会在高并发压力下丢失触发消息)
		commitTrigger: make(chan bool, 1),

		// 重要提示：缓冲通道
		// （未缓冲的通道可能会在高并发压力下丢失触发消息)
		snapshotTrigger: make(chan bool, 1),
		snapshotCh:      make(chan snapshotCmd),

		CurrentTerm:       0,           // 在第一次启动时初始化为0
		VotedFor:          voteForNull, // 启动时为null
		Log:               []LogEntry{},
		LastIncludedIndex: 0, // 初始化为0
		LastIncludedTerm:  0, // 初始化为0

		commitIndex:   0, // 初始化为0
		lastApplied:   0, // 初始化为0
		state:         Follower,
		electionAlarm: initElectionAlarm(),

		nextIndex:         nil, // 跟随者为空(nil)
		matchIndex:        nil, // 跟随者为空(nil)
		appendEntriesCh:   nil, // 跟随者为空(nil)
		installSnapshotCh: nil, // 跟随者为空(nil)
	}

	// 从崩溃前保存的状态初始化
	rf.readPersist(persister.ReadRaftState())

	// 将commitIndex和lastApplied设置为快照的LastIncludedIndex
	rf.commitIndex = rf.LastIncludedIndex
	rf.lastApplied = rf.LastIncludedIndex

	lastLogIndex, lastLogTerm := rf.lastLogIndexAndTerm()
	lablog.Debug(rf.me, lablog.Client, "Started at T:%d with (LII:%d LIT:%d), (LLI:%d LLT:%d)", rf.CurrentTerm, rf.LastIncludedIndex, rf.LastIncludedTerm, lastLogIndex, lastLogTerm)

	// 启动时钟goroutine以开始选举
	go rf.ticker()

	// 启动提交器goroutine以等待提交触发信号，提前lastApplied并将msg应用于客户端
	go rf.committer(applyCh, rf.commitTrigger)

	// 启动snapshoter goroutine以等待将通过snapshotCh发送的快照命令
	// 如果是领导者，则可能保留少量滞后的日志条目以使滞后的跟随者赶上
	// 这样领导者的快照就会延迟，并且每次更新leader的nextIndex时都将由snapshotTrigger触发
	go rf.snapshoter(rf.snapshotTrigger)

	return rf
}

// 返回CurrentTerm和此服务器是否认为自己是领导者。
func (rf *Raft) GetState() (term int, isLeader bool) {
	// 你的代码 (2A)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.CurrentTerm, rf.state == Leader
}

/************************* Helper *********************************************/

/*
		// 测试器不会在每个测试后停止 Raft 创建的 goroutine，但它会调用 Kill() 方法。
		您的代码可以使用 killed() 方法来检查是否已调用 Kill()。
		使用原子操作可以避免需要锁定。

	    问题在于长时间运行的 goroutine 使用内存并可能消耗 CPU 时间，
		导致后续测试失败并生成混乱的调试输出。
		任何具有长时间运行循环的 goroutine 都应该调用 killed() 方法来检查是否应该停止。
*/
func (rf *Raft) Kill() {
	// 设置标志以指示Raft节点已死亡
	atomic.StoreInt32(&rf.dead, 1)
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 终止所有长时间运行的goroutine
	// 停止EntriesAppender
	for _, c := range rf.appendEntriesCh {
		if c != nil {
			close(c)
		}
	}
	// 重要提示：不仅需要关闭通道，还需要重置appendEntriesCh以避免向已关闭的通道发送消息
	rf.appendEntriesCh = nil

	// 停止SnapshotInstaller
	for _, c := range rf.installSnapshotCh {
		if c != nil {
			close(c)
		}
	}
	// 重要提示：不仅需要关闭通道，还需要重置installSnapshotCh以避免向已关闭的通道发送消息
	rf.installSnapshotCh = nil

	// 停止Committer
	if rf.commitTrigger != nil {
		close(rf.commitTrigger)
	}
	// 重要提示：不仅需要关闭通道，还需要重置commitTrigger以避免向已关闭的通道发送消息
	rf.commitTrigger = nil

	// 停止Snapshoter
	close(rf.snapshotTrigger)
	// 重要提示：不仅需要关闭通道，还需要重置snapshotTrigger以避免向已关闭的通道发送消息
	rf.snapshotTrigger = nil
}

//通过raft结构体中的dead字段判断是否已经死亡  当被kill的时候会设置这个字段为0
func (rf *Raft) killed() bool {
	// 检查Raft节点是否已死亡
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// 强制升级为Follower状态，并重置其状态和领导者关联的所有状态
func (rf *Raft) stepDown(term int) {
	rf.state = Follower
	rf.CurrentTerm = term
	rf.VotedFor = voteForNull // 由于成为了Follower，投票将被重置
	rf.nextIndex = nil        // 由于成为了Follower，不再有成为Leader的机会，因此不需要维护NextIndex和MatchIndex
	rf.matchIndex = nil
	// 关闭所有通道并重置
	for _, c := range rf.appendEntriesCh {
		if c != nil {
			close(c)
		}
	}
	rf.appendEntriesCh = nil
	for _, c := range rf.installSnapshotCh {
		if c != nil {
			close(c)
		}
	}
	rf.installSnapshotCh = nil

	rf.persist()

	// 一旦成为Follower，触发任何延迟的快照
	select {
	case rf.snapshotTrigger <- true:
	default:
	}
}

// 获取最后一个日志条目的索引和任期，带锁
func (rf *Raft) lastLogIndexAndTerm() (index, term int) {
	index, term = rf.LastIncludedIndex, rf.LastIncludedTerm
	if l := len(rf.Log); l > 0 {
		//从当前节点的最后一条日志中获取索引和任期
		index, term = rf.Log[l-1].Index, rf.Log[l-1].Term
	}
	return
}

/************************* Election (2A) **************************************/

/*
// 在持有互斥锁的情况下，我的日志是否比其它节点的更为最新。
// Raft 通过比较日志中最后一个条目的索引和任期来确定两个日志哪个更为最新。
*/
func (rf *Raft) isMyLogMoreUpToDate(index int, term int) bool {
	// 如果日志具有不同任期的最后一个条目，则具有较晚任期的日志更为最新。
	myLastLogIndex, myLastLogTerm := rf.lastLogIndexAndTerm()
	if myLastLogTerm != term {
		return myLastLogTerm > term
	}

	// 如果日志的任期相同，则较长的日志更为最新
	return myLastLogIndex > index
}

// candidate 赢得选举，获得权力，同时持有互斥锁
func (rf *Raft) winElection() {
	rf.state = Leader

	// 重新初始化选举后的状态

	// 初始化为 leader 的最后一个日志索引 + 1
	lastLogIndex, _ := rf.lastLogIndexAndTerm()
	rf.nextIndex = make([]int, len(rf.peers))
	for i := range rf.nextIndex {
		rf.nextIndex[i] = lastLogIndex + 1
	}

	// 初始化为 0
	rf.matchIndex = make([]int, len(rf.peers))
	for i := range rf.matchIndex {
		rf.matchIndex[i] = 0
	}
	// 但是我知道我自己最高的日志条目
	rf.matchIndex[rf.me] = lastLogIndex

	// 为每个跟随者启动 entriesAppender goroutine，在当前任期中
	rf.appendEntriesCh = make([]chan int, len(rf.peers))
	for i := range rf.appendEntriesCh {
		if i != rf.me {
			rf.appendEntriesCh[i] = make(chan int, 1)
			go rf.entriesAppender(i, rf.appendEntriesCh[i], rf.CurrentTerm)
		}
	}

	// 为每个跟随者启动 snapshotInstaller goroutine，在当前任期中
	rf.installSnapshotCh = make([]chan int, len(rf.peers))
	for i := range rf.installSnapshotCh {
		if i != rf.me {
			rf.installSnapshotCh[i] = make(chan int, 1)
			go rf.snapshotInstaller(i, rf.installSnapshotCh[i], rf.CurrentTerm)
		}
	}

	// 启动 pacemaker 定期向每个跟随者发送心跳，在当前任期中
	go rf.pacemaker(rf.CurrentTerm)
}

type RequestVoteArgs struct {
	// 你的数据 (2A, 2B)。
	Term         int // 候选人的任期
	CandidateId  int // 请求投票的候选人
	LastLogIndex int // 候选人的最后一个日志条目的索引
	LastLogTerm  int // 候选人的最后一个日志条目的任期
}

type RequestVoteReply struct {
	// 你的数据 (2A)。
	Term        int  // 当前任期，用于候选人更新自己
	VoteGranted bool // true 表示候选人获得了选票
}

/********************** RequestVote RPC 处理程序 *******************************/

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.CurrentTerm
	reply.VoteGranted = false

	if args.Term < rf.CurrentTerm || args.CandidateId == rf.me || rf.killed() {
		// 回复给候选人和领导者的 RPC，如果 term < CurrentTerm，则返回 false
		// 跟随者只会回应其他服务器的请求
		return
	}

	if args.Term > rf.CurrentTerm {
		// 如果 RPC 请求的 term T > CurrentTerm：设置 CurrentTerm = T，转变为跟随者
		lablog.Debug(rf.me, lablog.Term, "C%d 请求的 RV 任期较高(%d > %d)，进行转变", args.CandidateId, args.Term, rf.CurrentTerm)
		rf.stepDown(args.Term)
	}

	//isMyLogMoreUpToDate会会检查任期，任期大的为最新，任期一样的看日志索引哪个最新
	lablog.Debug(rf.me, lablog.Vote, "C%d 请求投票，T%d", args.CandidateId, args.Term)
	if (rf.VotedFor == voteForNull || rf.VotedFor == args.CandidateId) &&
		!rf.isMyLogMoreUpToDate(args.LastLogIndex, args.LastLogTerm) {
		// 如果 VotedFor 是 null 或者 candidateId，
		// 并且候选人的日志至少与接收者的日志一样新，
		// 授予选票
		lablog.Debug(rf.me, lablog.Vote, "授予 C%d 在 T%d 的选票", args.CandidateId, args.Term)
		reply.VoteGranted = true
		rf.VotedFor = args.CandidateId

		// 重要：
		// 采纳 [students-guide-to-raft](https://thesquareplanet.com/blog/students-guide-to-raft/#livelocks) 的建议
		// 引用：
		//   如果 a)...；b)...；c) 你授予另一个对等体选票，重新启动你的选举定时器
		rf.electionAlarm = nextElectionAlarm()

		rf.persist()
	}
}

/********************** RequestVote RPC caller ********************************/

// 创建RequestVote RPC的参数，互斥锁已握住
func (rf *Raft) constructRequestVoteArgs() *RequestVoteArgs {
	lastLogIndex, lastLogTerm := rf.lastLogIndexAndTerm()
	return &RequestVoteArgs{
		Term:         rf.CurrentTerm,
		CandidateId:  rf.me,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
}

/*
// 向某台服务器发送RequestVote RPC，server为目标服务器在rf.peers[]中的索引，args为RPC请求参数
// reply为RPC响应结果，使用&reply传递地址
// Call()发送请求并等待响应，在超时时间内等待响应，如果在超时时间内收到响应，则返回true；否则返回false
// 如果handler函数在服务器端没有返回，Call()将会被阻塞，因此不需要实现自己的超时机制
// 更多详情请查看../labrpc/labrpc.go中的注释内容
*/
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	lablog.Debug(rf.me, lablog.Vote, "T%d -> S%d Sending RV, LLI:%d LLT:%d", args.Term, server, args.LastLogIndex, args.LastLogTerm)
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// requestVote协程向一个节点发送RequestVote RPC并处理响应结果
//向其他节点的某个节点请求投票
func (rf *Raft) requestVote(server int, term int, args *RequestVoteArgs, grant chan<- bool) {
	//记录是否给了投票
	granted := false
	//会在函数返回前执行，这个匿名函数是将变量 granted 的值通过通道 grant 传回给调用方
	defer func() { grant <- granted }()

	rf.mu.Lock()
	if rf.state != Candidate || rf.CurrentTerm != term || rf.killed() {
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()

	reply := &RequestVoteReply{}
	r := rf.sendRequestVote(server, args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 再次检查所有相关的假设条件
	// - 仍然是候选人状态
	// - 实例没有被关闭
	if rf.state != Candidate || rf.killed() {
		return
	}

	if !r {
		lablog.Debug(rf.me, lablog.Drop, "-> S%d RV been dropped: {T:%d LLI:%d LLT:%d}", server, args.Term, args.LastLogIndex, args.LastLogTerm)
		return
	}

	if reply.Term > rf.CurrentTerm {
		// If RPC response contains term T > CurrentTerm: set CurrentTerm = T, convert to follower
		lablog.Debug(rf.me, lablog.Term, "RV <- S%d Term is higher(%d > %d), following", server, reply.Term, rf.CurrentTerm)
		rf.stepDown(reply.Term)
		return
	}

	// 重要提示：
	// 采纳来源自 students-guide-to-raft
	// 引用如下：
	// 从我们的经验来看，最简单的方法是首先记录回复中的术语（可能比您当前的术语更高)，
	// 然后将当前术语与您在原始RPC中发送的术语进行比较。如果两者不同，则放弃回复并返回。
	// 如果两个术语相同，才应继续处理回复。
	if rf.CurrentTerm != term {
		return
	}

	granted = reply.VoteGranted
	lablog.Debug(rf.me, lablog.Vote, "<- S%d Got Vote: %t, at T%d", server, granted, term)
}

// collectVote协程从同行收集投票
func (rf *Raft) collectVote(term int, grant <-chan bool) {
	cnt := 1
	done := false
	for i := 0; i < len(rf.peers)-1; i++ {
		if <-grant {
			cnt++
		}
		// 收到大多数服务器的选票：成为领导者
		if !done && cnt >= len(rf.peers)/2+1 {
			done = true
			rf.mu.Lock()
			// 重新检查所有相关的假设
			// - 仍然是候选人
			// - rf.CurrentTerm自从决定成为候选人以来没有改变
			// - 安装未被杀死
			if rf.state != Candidate || rf.CurrentTerm != term || rf.killed() {
				rf.mu.Unlock()
			} else {
				rf.winElection()
				lablog.Debug(rf.me, lablog.Leader, "Achieved Majority for T%d, converting to Leader, NI:%v, MI:%v", rf.CurrentTerm, rf.nextIndex, rf.matchIndex)
				rf.mu.Unlock()
			}
		}
	}
}

// ticker协程如果此peer最近没有收到心跳，则启动新选举。
func (rf *Raft) ticker() {
	//time.Duration是Go语言中的时间类型。它表示持续时间，可以用来表示时间段或时间间隔。
	var sleepDuration time.Duration
	// 长期运行的单例协程，应检查是否被杀死
	for !rf.killed() {
		// 您的代码在此处检查是否应该启动领导者选举并使用随机睡眠时间进行随机化
		rf.mu.Lock()
		if rf.state == Leader {
			// 领导者每次重置自己的选举超时警报
			rf.electionAlarm = nextElectionAlarm()
			sleepDuration = time.Until(rf.electionAlarm)
			rf.mu.Unlock()
		} else {
			lablog.Debug(rf.me, lablog.Timer, "Not Leader, checking election timeout")
			if rf.electionAlarm.After(time.Now()) {
				// 没有达到选举超时，进入睡眠状态
				sleepDuration = time.Until(rf.electionAlarm)
				rf.mu.Unlock()
			} else {
				// 如果在当前领导者没有接收到AppendEntries RPC或将选票授予候选人的情况下超过选举超时：
				// 转换为候选人，开始新选举

				// 在转化为候选者时，启动选举：
				// - 增加CurrentTerm
				rf.CurrentTerm++
				term := rf.CurrentTerm
				lablog.Debug(rf.me, lablog.Term, "改变任期，启动选举, 目前的选举任期号为 T:%d", term)
				// - 更改为候选人
				rf.state = Candidate
				// - 为自己投票
				rf.VotedFor = rf.me

				rf.persist()

				// - 重置选举定时器
				lablog.Debug(rf.me, lablog.Timer, "Resetting ELT because election")
				rf.electionAlarm = nextElectionAlarm()
				sleepDuration = time.Until(rf.electionAlarm)

				args := rf.constructRequestVoteArgs()
				rf.mu.Unlock()

				grant := make(chan bool)
				// - 向所有其他服务器发送RequestVote RPC
				for i := range rf.peers {
					if i != rf.me {
						go rf.requestVote(i, term, args, grant)
					}
				}

				// 候选人收集当前任期的选票
				go rf.collectVote(args.Term, grant)
			}
		}

		lablog.Debug(rf.me, lablog.Timer, "Ticker going to sleep for %d ms", sleepDuration.Milliseconds())
		time.Sleep(sleepDuration)
	}
}

/************************* 日志 (2B) *******************************************/

type AppendEntriesArgs struct {
	Term         int        // 领导者的任期
	LeaderId     int        // 以便跟随者可以重定向客户端
	PrevLogIndex int        // 新日志条目之前的日志条目索引
	PrevLogTerm  int        // prevLogIndex 条目的任期
	Entries      []LogEntry // 要存储的日志条目（心跳时为空)
	LeaderCommit int        // 领导者的提交索引
}

type AppendEntriesReply struct {
	Term    int  // CurrentTerm，用于领导者更新自身
	Success bool // 如果跟随者包含与 prevLogIndex 和 prevLogTerm 匹配的条目，则为 true
	// 优化

	XTerm  int // 冲突条目中的任期（如果存在)
	XIndex int // 第一个具有 XTerm 任期的条目的索引（如果存在)
	XLen   int // 日志长度
}

/********************** AppendEntries RPC 处理程序 *****************************/

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Term = rf.CurrentTerm
	reply.Success = false
	////////////////////////////////////////任期问题
	if args.Term < rf.CurrentTerm || args.LeaderId == rf.me || rf.killed() {
		// 响应候选人和领导者的 RPC
		// 跟随者只响应其他服务器的请求
		// 如果 term < CurrentTerm，则回复 false
		return
	}

	if args.Term > rf.CurrentTerm {
		// 如果 RPC 请求包含的任期 T > CurrentTerm：将 CurrentTerm = T，转换为跟随者
		lablog.Debug(rf.me, lablog.Term, "S%d %s 请求的任期较高（%d > %d),进行跟随", args.LeaderId, intentOfAppendEntriesRPC(args), args.Term, rf.CurrentTerm)
		rf.stepDown(args.Term)
	}

	if rf.state == Candidate && args.Term >= rf.CurrentTerm {
		// 在等待投票时,候选人可能会收到来自另一台服务器的 AppendEntries RPC,声称自己是领导者。
		// 如果领导者的任期至少与候选人的当前任期一样大,则候选人将领导者视为合法,并返回跟随者状态
		lablog.Debug(rf.me, lablog.Term, "我是候选人,S%d %s 请求的任期 %d >= %d,进行跟随", args.LeaderId, intentOfAppendEntriesRPC(args), args.Term, rf.CurrentTerm)
		rf.stepDown(args.Term)
	}

	lablog.Debug(rf.me, lablog.Timer, "重置选举超时,从 L%d 接收到 %s,在 T%d", args.LeaderId, intentOfAppendEntriesRPC(args), args.Term)
	rf.electionAlarm = nextElectionAlarm()
	///////////////////////////////////////////////可以不优化，说明自己的最大日志
	// 如果日志不包含与 prevLogIndex 的任期匹配的条目,则回复 false
	lastLogIndex, _ := rf.lastLogIndexAndTerm()
	if lastLogIndex < args.PrevLogIndex {
		// 优化：快速日志回滚,跟随者的日志太短
		reply.XLen = lastLogIndex + 1
		return
	}
	///////////////////////////////////////////////有足够日志，接下里就是正常比对////////////////////////////
	// lastLogIndex >= args.PrevLogIndex，跟随者的日志足够长，可以覆盖 args.PrevLogIndex
	var prevLogTerm int
	switch {
	case args.PrevLogIndex == rf.LastIncludedIndex:
		prevLogTerm = rf.LastIncludedTerm
	case args.PrevLogIndex < rf.LastIncludedIndex:
		// 跟随者已经提交了 PrevLogIndex 日志 =>
		// PrevLogIndex 的一致性检查已经完成 =>
		// 可以从 rf.LastIncludedIndex 开始进行日志一致性检查
		args.PrevLogIndex = rf.LastIncludedIndex
		prevLogTerm = rf.LastIncludedTerm
		// 将 args.Entries 截断，以从 LastIncludedIndex 开始
		sameEntryInArgsEntries := false
		for i := range args.Entries {
			if args.Entries[i].Index == rf.LastIncludedIndex && args.Entries[i].Term == rf.LastIncludedTerm {
				sameEntryInArgsEntries = true
				args.Entries = args.Entries[i+1:]
				break
			}
		}
		if !sameEntryInArgsEntries {
			// 在 args.Entries 中未找到 LastIncludedIndex 的日志条目 =>
			// args.Entries 全部被 LastIncludedIndex 覆盖 =>
			// args.Entries 全部已提交 =>
			// 可以将 args.Entries 截断为空切片
			args.Entries = make([]LogEntry, 0)
		}
	default:
		// args.PrevLogIndex > rf.LastIncludedIndex
		prevLogTerm = rf.Log[args.PrevLogIndex-rf.LastIncludedIndex-1].Term
	}

	if prevLogTerm != args.PrevLogTerm {
		// 优化：快速日志回滚：设置冲突条目的任期和具有该任期的第一个条目的索引
		reply.XTerm = prevLogTerm
		for i := args.PrevLogIndex - rf.LastIncludedIndex - 1; i > -1; i-- {
			reply.XIndex = rf.Log[i].Index
			if rf.Log[i].Term != prevLogTerm {
				break
			}
		}
		return
	}
	//////////////////////////////////////////////////////最好的情况
	// 如果跟随者包含与 prevLogIndex 和 prevLogTerm 匹配的条目，则为 true
	reply.Success = true

	if len(args.Entries) > 0 {
		// 如果现有条目与新条目冲突（索引相同但任期不同)，则删除现有条目及其后续条目
		lablog.Debug(rf.me, lablog.Info, "接收到来自 S%d 的日志条目：%v，在 T%d", args.LeaderId, args.Entries, args.Term)
		existingEntries := rf.Log[args.PrevLogIndex-rf.LastIncludedIndex:]
		var i int
		needPersist := false
		for i = 0; i < labutil.Min(len(existingEntries), len(args.Entries)); i++ {
			if existingEntries[i].Term != args.Entries[i].Term {
				lablog.Debug(rf.me, lablog.Info, "丢弃冲突的日志条目：%v", rf.Log[args.PrevLogIndex-rf.LastIncludedIndex+i:])
				rf.Log = rf.Log[:args.PrevLogIndex-rf.LastIncludedIndex+i]
				needPersist = true
				break
			}
		}
		if i < len(args.Entries) {
			// 追加任何尚未在日志中的新条目
			lablog.Debug(rf.me, lablog.Info, "追加新的日志条目：%v，从索引 %d 开始", args.Entries[i:], i)
			rf.Log = append(rf.Log, args.Entries[i:]...)
			needPersist = true
		}

		if needPersist {
			rf.persist()
		}
	}

	if args.LeaderCommit > rf.commitIndex {
		// 如果 leaderCommit > commitIndex，则设置 commitIndex = min(leaderCommit, 最后一个新条目的索引)
		lastLogIndex, _ := rf.lastLogIndexAndTerm()
		rf.commitIndex = labutil.Min(args.LeaderCommit, lastLogIndex)
		// 准备提交
		select {
		case rf.commitTrigger <- true:
		default:
		}
	}
}

/********************** AppendEntries RPC caller ******************************/
// 创建AppendEntries RPC参数，同时持有互斥锁
// 重要提示：返回nil表示需要安装快照
func (rf *Raft) constructAppendEntriesArgs(server int) *AppendEntriesArgs {
	prevLogIndex := rf.nextIndex[server] - 1
	prevLogTerm := rf.LastIncludedTerm
	if i := prevLogIndex - rf.LastIncludedIndex - 1; i > -1 {
		prevLogTerm = rf.Log[i].Term
	}

	var entries []LogEntry
	if lastLogIndex, _ := rf.lastLogIndexAndTerm(); lastLogIndex <= prevLogIndex {
		// 该追随者已经拥有了领导者的所有日志，无需发送日志条目
		entries = nil
	} else if prevLogIndex >= rf.LastIncludedIndex {
		// 发送从nextIndex开始的日志条目的AppendEntries RPC
		newEntries := rf.Log[prevLogIndex-rf.LastIncludedIndex:]
		entries = make([]LogEntry, len(newEntries))
		// 避免数据竞争
		copy(entries, newEntries)
	} else {
		// 重要提示：领导者的日志太短，无法追加日志条目，需要安装快照
		return nil
	}

	return &AppendEntriesArgs{
		Term:         rf.CurrentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: rf.commitIndex,
	}
}

// 向服务器发送AppendEntries RPC
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	lablog.Debug(rf.me, dTopicOfAppendEntriesRPC(args, lablog.Info), "T%d -> S%d 发送 %s, PLI:%d PLT:%d LC:%d - %v", args.Term, server, intentOfAppendEntriesRPC(args), args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Entries)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// appendEntries Go程向一个对等体发送AppendEntries RPC并处理回复
func (rf *Raft) appendEntries(server int, term int, args *AppendEntriesArgs, serialNo int) {
	reply := &AppendEntriesReply{}
	r := rf.sendAppendEntries(server, args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 重新检查所有相关的假设
	// - 仍然是领导者
	// - 实例未被关闭
	if rf.state != Leader || rf.killed() {
		return
	}

	rpcIntent := intentOfAppendEntriesRPC(args)

	if !r {
		if rpcIntent == "HB" {
			// 优化：不要重试心跳RPC
			return
		}
		if args.PrevLogIndex < rf.nextIndex[server]-1 {
			// 优化：此AppendEntries RPC已过时，不要重试
			return
		}
		// 当服务器没有回复时重试
		select {
		case rf.appendEntriesCh[server] <- serialNo: // 使用serialNo重试
			lablog.Debug(rf.me, lablog.Drop, "-> S%d %s 被丢弃：{T:%d PLI:%d PLT:%d LC:%d log长度:%d}，重试", server, rpcIntent, args.Term, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, len(args.Entries))
		default:
		}
		return
	}

	lablog.Debug(rf.me, dTopicOfAppendEntriesRPC(args, lablog.Log), "%s <- S%d 回复: %+v", rpcIntent, server, *reply)

	if reply.Term > rf.CurrentTerm {
		// 如果RPC响应包含大于当前任期的任期T：设置CurrentTerm = T，并转换为追随者
		lablog.Debug(rf.me, lablog.Term, "%s <- S%d 任期更高(%d > %d)，转为追随者", rpcIntent, server, reply.Term, rf.CurrentTerm)
		rf.stepDown(reply.Term)
		rf.electionAlarm = nextElectionAlarm()
		return
	}

	// 重要提示：
	// 采纳[students-guide-to-raft](https://thesquareplanet.com/blog/students-guide-to-raft/#term-confusion)中#Term-confusion部分的建议
	// 引用：
	// 经验表明，最简单的方法是首先记录回复中的任期（可能比当前任期更高），
	// 然后将当前任期与您在原始RPC中发送的任期进行比较。
	// 如果两者不同，则丢弃回复并返回。
	// 只有两个任期相同时才继续处理回复。
	if rf.CurrentTerm != term {
		return
	}

	if reply.Success {
		// 如果成功：更新追随者的nextIndex和matchIndex
		// 如果成功，nextIndex应该只增加
		oldNextIndex := rf.nextIndex[server]
		// 如果网络对RPC回复进行重新排序，并且要更新的nextIndex < 对等体的当前nextIndex，
		// 不要将nextIndex更新为较小的值（以避免重新发送追随者已经拥有的日志条目）
		// matchIndex也是如此
		rf.nextIndex[server] = labutil.Max(args.PrevLogIndex+len(args.Entries)+1, rf.nextIndex[server])
		rf.matchIndex[server] = labutil.Max(args.PrevLogIndex+len(args.Entries), rf.matchIndex[server])
		lablog.Debug(rf.me, dTopicOfAppendEntriesRPC(args, lablog.Log), "%s RPC -> S%d 成功，更新NI:%v，MI:%v", rpcIntent, server, rf.nextIndex, rf.matchIndex)

		// 更新了matchIndex，可能有一些日志条目可以提交，进行检查
		go rf.checkCommit(term)

		if rf.nextIndex[server] > oldNextIndex {
			// 更新了nextIndex，告诉快照器检查是否可以处理任何延迟的快照命令
			select {
			case rf.snapshotTrigger <- true:
			default:
			}
		}

		return
	}

	// 如果AppendEntries因为日志不一致而失败：
	// 减小nextIndex并重试
	needToInstallSnapshot := false
	// 优化：快速日志回滚
	if reply.XLen != 0 && reply.XTerm == 0 {
		// 追随者的日志太短
		rf.nextIndex[server] = reply.XLen
	} else {
		var entryIndex, entryTerm int
		for i := len(rf.Log) - 1; i >= -1; i-- {
			if i < 0 {
				entryIndex, entryTerm = rf.lastLogIndexAndTerm()
			} else {
				entryIndex, entryTerm = rf.Log[i].Index, rf.Log[i].Term
			}

			if entryTerm == reply.XTerm {
				// 领导者的日志有XTerm
				rf.nextIndex[server] = entryIndex + 1
				break
			}
			if entryTerm < reply.XTerm {
				// 领导者的日志没有XTerm
				rf.nextIndex[server] = reply.XIndex
				break
			}

			if i < 0 {
				// 领导者的日志太短，需要向追随者安装快照
				needToInstallSnapshot = true
				rf.nextIndex[server] = rf.LastIncludedIndex + 1
			}
		}
	}

	if needToInstallSnapshot || rf.nextIndex[server] <= rf.LastIncludedIndex {
		// 领导者的日志太短，需要向追随者安装快照
		select {
		case rf.installSnapshotCh[server] <- 0:
		default:
		}
	} else {
		// 尝试追加更多的先前日志条目
		select {
		case rf.appendEntriesCh[server] <- 0:
		default:
		}
	}
}

// entriesAppender是一个用于向服务器调用AppendEntries RPC的单点goroutine。
// 当选举胜出后，领导者会为其任期内的每个跟随者启动此goroutine，
// 一旦下台或终止，该goroutine将终止。
func (rf *Raft) entriesAppender(server int, ch <-chan int, term int) {
	i := 1 // 序列号
	for !rf.killed() {
		serialNo, ok := <-ch
		if !ok {
			return // 通道关闭，应终止此goroutine
		}
		rf.mu.Lock()
		// 重新检查
		if rf.state != Leader || rf.CurrentTerm != term || rf.killed() {
			rf.mu.Unlock()
			return
		}

		args := rf.constructAppendEntriesArgs(server)
		if args == nil {
			// 无法附加日志条目，转而安装快照
			select {
			case rf.installSnapshotCh[server] <- 0:
				lablog.Debug(rf.me, lablog.Snap, "-> S%d 无法附加日志条目，转向安装快照", server)
			default:
			}
			rf.mu.Unlock()
			continue
		}
		rf.mu.Unlock()

		if serialNo == 0 || // 新的AppendEntries RPC调用
			serialNo >= i { // 是重试，需要检查序列号以忽略过时的调用

			rf.appendEntries(server, term, args, i) // 提供i作为序列号，以便在重试时传回
			// 增加此entriesAppender的序列号
			i++
		}
	}
}

// 使用Raft的服务（例如k/v服务器）希望开始就下一个要附加到Raft日志的命令达成一致。
// 如果该服务器不是领导者，则返回false。否则，开始达成一致并立即返回。
// 不能保证该命令将被提交到Raft日志中，因为领导者可能会失败或丢失选举。
// 即使Raft实例被终止，此函数也应该正常返回。
//
// 第一个返回值是命令在如果被提交时将出现的索引。
// 第二个返回值是当前的任期。第三个返回值是如果该服务器认为自己是领导者则为true。
func (rf *Raft) Start(command interface{}) (index int, term int, isLeader bool) {
	// 在这里编写你的代码（2B）。
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.CurrentTerm
	if rf.state != Leader || rf.killed() {
		return
	}

	index = rf.nextIndex[rf.me]
	isLeader = true

	// 如果从客户端收到命令：将条目附加到本地日志中
	rf.Log = append(rf.Log, LogEntry{
		Index:   index,
		Term:    term,
		Command: command,
	})
	rf.nextIndex[rf.me]++
	rf.matchIndex[rf.me] = index
	lablog.Debug(rf.me, lablog.Log2, "收到日志：%v，NI：%v，MI：%v", rf.Log[len(rf.Log)-1], rf.nextIndex, rf.matchIndex)

	rf.persist()

	// 触发所有跟随者的entriesAppender，请求达成一致
	for i, c := range rf.appendEntriesCh {
		if i != rf.me {
			select {
			case c <- 0:
			default:
			}
		}
	}

	return
}

// pacemaker是一个定期向所有跟随者广播心跳的goroutine。
// 当选举时：向每个服务器发送初始的空AppendEntries RPC（心跳）。
// 在空闲期间重复以防止选举超时。
func (rf *Raft) pacemaker(term int) {
	// 领导者的pacemaker仅持续其当前任期
	for !rf.killed() {
		rf.mu.Lock()
		// 在广播之前每次重新检查状态和任期
		if rf.state != Leader || rf.CurrentTerm != term {
			rf.mu.Unlock()
			// 本任期的pacemaker结束
			return
		}
		// 向每个服务器发送心跳
		lablog.Debug(rf.me, lablog.Timer, "领导者在T%d，广播心跳", term)
		for i, c := range rf.appendEntriesCh {
			if i != rf.me {
				select {
				case c <- 0:
				default:
				}
			}
		}
		rf.mu.Unlock()

		// 在空闲期间重复以防止选举超时
		time.Sleep(heartbeatInterval * time.Millisecond)
	}
}

// checkCommit是一个检查是否有可提交的日志条目的goroutine，
// 如果有，则进行提交（在单独的goroutine中）。
func (rf *Raft) checkCommit(term int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 重新检查状态和任期
	if rf.state != Leader || rf.CurrentTerm != term || rf.killed() {
		return
	}

	// 领导者的commitIndex已经是最后一个了，没有可提交的内容
	lastLogIndex, _ := rf.lastLogIndexAndTerm()
	if rf.commitIndex >= lastLogIndex {
		return
	}

	// 如果存在一个N使得：
	// - N > commitIndex
	// - 大多数matchIndex[i] ≥ N
	// - log[N].term == CurrentTerm：
	// 设置commitIndex = N
	oldCommitIndex := rf.commitIndex
	for n := rf.commitIndex + 1; n < len(rf.Log)+rf.LastIncludedIndex+1; n++ {
		if rf.Log[n-rf.LastIncludedIndex-1].Term == term {
			cnt := 1
			for i := range rf.peers {
				if i != rf.me && rf.matchIndex[i] >= n {
					cnt++
				}
			}
			if cnt >= len(rf.peers)/2+1 {
				lablog.Debug(rf.me, lablog.Commit, "提交达到多数派，将CI从%d设置为%d", rf.commitIndex, n)
				rf.commitIndex = n
			}
		}
	}

	if oldCommitIndex != rf.commitIndex {
		// 准备进行提交
		select {
		case rf.commitTrigger <- true:
		default:
		}
	}
}

// committer是一个后台goroutine，用于比较commitIndex和lastApplied，
// 如果是安全的，则增加lastApplied，并且如果有已应用的日志，则通过applyCh发送给客户端。
func (rf *Raft) committer(applyCh chan<- ApplyMsg, triggerCh chan bool) {
	defer func() {
		// 重要：关闭通道以避免资源泄漏
		close(applyCh)
		// 重要：清空commitTrigger以避免goroutine资源泄漏
		for i := 0; i < len(triggerCh); i++ {
			<-triggerCh
		}
	}()

	// 长期运行的独立goroutine，需要检查是否被终止
	for !rf.killed() {
		isCommit, ok := <-triggerCh // 等待信号

		if !ok {
			return
		}

		rf.mu.Lock()

		if !isCommit {
			// 重新启用commitTrigger以准备接收提交信号
			rf.commitTrigger = triggerCh
			// 收到来自领导者的快照
			data := rf.persister.ReadSnapshot()
			if rf.LastIncludedIndex == 0 || len(data) == 0 {
				// 快照数据无效
				rf.mu.Unlock()
				continue
			}

			// 将快照发送回上层服务
			applyMsg := ApplyMsg{
				CommandValid:  false,
				SnapshotValid: true,
				Snapshot:      data,
				SnapshotIndex: rf.LastIncludedIndex,
				SnapshotTerm:  rf.LastIncludedTerm,
			}
			rf.mu.Unlock()

			applyCh <- applyMsg
			continue
		}

		// 重要：如果有等待发送回上层服务的快照，
		// 则应尽快应用快照，并且避免应用更多的日志条目。
		// 否则，尝试尽可能多地应用日志条目
		for rf.commitTrigger != nil && rf.commitIndex > rf.lastApplied {
			// 如果commitIndex > lastApplied：
			// 增加lastApplied，
			// 将log[lastApplied]应用于状态机
			rf.lastApplied++
			logEntry := rf.Log[rf.lastApplied-rf.LastIncludedIndex-1]
			lablog.Debug(rf.me, lablog.Client, "CI：%d > LA：%d，应用日志：%s", rf.commitIndex, rf.lastApplied-1, logEntry)

			// 在发送ApplyMsg到applyCh之前释放互斥锁
			rf.mu.Unlock()

			applyCh <- ApplyMsg{
				CommandValid: true,
				Command:      logEntry.Command,
				CommandIndex: logEntry.Index,
			}

			// 重新加锁
			rf.mu.Lock()
		}

		rf.mu.Unlock()
	}
}

/************************* 持久化（2C）***************************************/

// 将raft实例的状态以字节形式获取，需要持有互斥锁
func (rf *Raft) raftState() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if e.Encode(rf.CurrentTerm) != nil ||
		e.Encode(rf.VotedFor) != nil ||
		e.Encode(rf.Log) != nil ||
		e.Encode(rf.LastIncludedIndex) != nil ||
		e.Encode(rf.LastIncludedTerm) != nil {
		return nil
	}
	return w.Bytes()
}

// 将Raft的持久化状态保存到稳定存储中，需要持有互斥锁
// 在崩溃和重启后可以检索到之前保存的状态。
// 请参阅论文中的图2，了解应该持久化什么内容。
func (rf *Raft) persist() {
	// 你的代码（2C）。
	if data := rf.raftState(); data == nil {
		lablog.Debug(rf.me, lablog.Error, "写入持久化失败")
	} else {
		lastLogIndex, lastLogTerm := rf.lastLogIndexAndTerm()
		lablog.Debug(rf.me, lablog.Persist, "保存状态 T：%d VF：%d，（LII：%d LIT：%d），（LLI：%d LLT：%d）", rf.CurrentTerm, rf.VotedFor, rf.LastIncludedIndex, rf.LastIncludedTerm, lastLogIndex, lastLogTerm)
		rf.persister.SaveRaftState(data)
	}
}

// 恢复之前持久化的状态。
func (rf *Raft) readPersist(data []byte) {
	if len(data) == 0 { // 从头开始没有任何状态
		return
	}
	// 你的代码（2C）。
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm, votedFor, lastIncludedIndex, lastIncludedTerm int
	var logs []LogEntry
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&logs) != nil ||
		d.Decode(&lastIncludedIndex) != nil ||
		d.Decode(&lastIncludedTerm) != nil {
		lablog.Debug(rf.me, lablog.Error, "读取损坏的持久化状态")
		return
	}
	rf.CurrentTerm = currentTerm
	rf.VotedFor = votedFor
	rf.Log = logs
	rf.LastIncludedIndex = lastIncludedIndex
	rf.LastIncludedTerm = lastIncludedTerm
}

/************************* Log Compaction (2D) ********************************/

/************************* 日志压缩 (2D) ********************************/

// 如果服务收到更近的信息，那么服务希望切换到快照。只有在此情况下才进行切换。
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {
	// 在这里编写代码（2D）。
	// 先前，该实验建议您实现一个名为CondInstallSnapshot的函数，
	// 以避免在应用通道（applyCh）上协调快照和日志条目的要求。
	// 这个遗留的API接口仍然存在，但我们建议您只返回true。
	return true
}

// 服务表示已创建快照，其中包含了截止到指定索引的所有信息。
// 这意味着服务不再需要截止到该索引的日志条目。
// Raft 应该尽可能地修剪日志。
//kv中调用
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// 在这里编写代码（2D）。
	// 不需要立即处理
	// 发送到快照处理后台goroutine的队列中
	select {
	case rf.snapshotCh <- snapshotCmd{index, snapshot}:
	default:
	}
}

// 在持有互斥锁的情况下检查是否应该暂停索引处的快照
// 领导者：如果有任何一个追随者可能滞后（在一个小范围内，但不要太远，由leaderKeepLogAmount限制），则暂停；否则，不暂停。
func (rf *Raft) shouldSuspendSnapshot(index int) (r bool) {
	r = false
	if rf.state != Leader {
		return
	}
	for i := range rf.peers {
		if distance := index - rf.nextIndex[i]; distance >= 0 && distance <= leaderKeepLogAmount {
			// 如果有任何一个追随者滞后，领导者会保留一小部分尾随日志一段时间
			r = true
			break
		}
	}
	return
}

// 在持有互斥锁的情况下保存快照和Raft状态
func (rf *Raft) saveStateAndSnapshot(snapshot []byte) {
	if data := rf.raftState(); data == nil {
		lablog.Debug(rf.me, lablog.Error, "保存快照失败")
	} else {
		lastLogIndex, lastLogTerm := rf.lastLogIndexAndTerm()
		lablog.Debug(rf.me, lablog.Snap, "保存状态：T：%d VF：%d，（LII：%d LIT：%d），（LLI：%d LLT：%d）和快照", rf.CurrentTerm, rf.VotedFor, rf.LastIncludedIndex, rf.LastIncludedTerm, lastLogIndex, lastLogTerm)
		rf.persister.SaveStateAndSnapshot(data, snapshot)
	}
}

// 快照处理后台goroutine作为长期运行的单例goroutine处理快照命令
func (rf *Raft) snapshoter(triggerCh <-chan bool) {
	// 仅保留一个快照命令
	var index int
	var snapshot []byte
	// 可能设置为nil以阻止从snapshotCh接收，以防止当前快照被覆盖
	cmdCh := rf.snapshotCh
	// 长期运行的单例goroutine，应检查是否已终止
	for !rf.killed() {
		select {
		case cmd := <-cmdCh:
			index, snapshot = cmd.index, cmd.snapshot
		case _, ok := <-triggerCh:
			if !ok {
				return
			}
		}

		rf.mu.Lock()
		shouldSuspend := rf.shouldSuspendSnapshot(index)

		if cmdCh == nil {
			// 阻塞接收snapshotCh =>
			// 有一个延迟的快照等待处理和持久化，
			// 重新检查是否可以处理此延迟的快照
			if shouldSuspend {
				// 不行 => 等待新的触发信号
				rf.mu.Unlock()
				continue
			}
			// 可以 => 恢复snapshotCh，准备接收新的快照命令
			cmdCh = rf.snapshotCh
		}

		switch {
		case index <= rf.LastIncludedIndex:
			// 此快照已被LastIncludedIndex覆盖 =>
			// 此快照已过时 =>
			// 忽略
		case shouldSuspend:
			// 快照是最新的，但应该暂停，
			// 因此阻塞从snapshotCh接收，并等待触发信号
			cmdCh = nil
		default:
			// 快照是最新的，可以处理保存

			// 将索引处的任期记录为LastIncludedTerm以进行持久化
			rf.LastIncludedTerm = rf.Log[index-rf.LastIncludedIndex-1].Term
			// 丢弃从索引到索引的旧日志条目
			rf.Log = rf.Log[index-rf.LastIncludedIndex:]
			// 记录索引为LastIncludedIndex以进行持久化
			rf.LastIncludedIndex = index

			rf.saveStateAndSnapshot(snapshot)
		}

		rf.mu.Unlock()
	}
}

type InstallSnapshotArgs struct {
	Term              int    // 领导者的任期
	LeaderId          int    // 以便追随者可以重定向客户端
	LastIncludedIndex int    // 快照替换了截至此索引的所有条目
	LastIncludedTerm  int    // LastIncludedIndex的任期
	Offset            int    // 快照文件中的块位置的字节偏移量（未使用）
	Data              []byte // 快照块的原始字节，从Offset开始
	Done              bool   // 如果这是最后一个块，则为true
}

/*************** 翻译结束 ***************/

type InstallSnapshotReply struct {
	Term int // 当前任期，用于领导者更新自己
}

/********************** InstallSnapshot RPC处理程序 ***************************/

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.CurrentTerm

	if args.Term < rf.CurrentTerm || args.LeaderId == rf.me || rf.killed() {
		// 响应候选人和领导者的RPC请求
		// 跟随者只回应其他服务器的请求
		// 如果任期小于CurrentTerm，则立即回应
		return
	}

	if args.Term > rf.CurrentTerm {
		// 如果RPC请求包含大于CurrentTerm的任期T：设置CurrentTerm = T，转变为跟随者
		lablog.Debug(rf.me, lablog.Term, "S%d的IS请求的任期更高(%d > %d)，切换为跟随者", args.LeaderId, args.Term, rf.CurrentTerm)
		rf.stepDown(args.Term)
	}

	lablog.Debug(rf.me, lablog.Timer, "重置ELT，接收到L%d的IS，任期为T%d", args.LeaderId, args.Term)
	rf.electionAlarm = nextElectionAlarm()

	if args.LastIncludedIndex <= rf.LastIncludedIndex || // 过时的快照
		args.LastIncludedIndex <= rf.lastApplied { // 重要：已经应用了比快照更多的日志条目，不需要安装此快照，否则将错过其中的一些日志条目
		return
	}

	// 假设args.Offset == 0并且args.Done == true
	lablog.Debug(rf.me, lablog.Snap, "从S%d接收到快照，任期为T%d，(LII:%d LIT:%d)", args.LeaderId, args.Term, args.LastIncludedIndex, args.LastIncludedTerm)

	// 快照被接受，开始安装

	rf.LastIncludedIndex = args.LastIncludedIndex
	rf.LastIncludedTerm = args.LastIncludedTerm
	// 更新commitIndex和lastApplied
	rf.commitIndex = labutil.Max(args.LastIncludedIndex, rf.commitIndex)
	rf.lastApplied = args.LastIncludedIndex

	defer func() {
		// 保存快照文件
		// 丢弃任何现有或部分较小索引的快照（由持久化器完成）
		rf.saveStateAndSnapshot(args.Data)

		if rf.commitTrigger != nil {
			// 准备发送快照到服务
			// 不能丢失此触发信号，必须等待通道发送完成，
			// 因此不能使用选择-默认方案
			go func(ch chan<- bool) { ch <- false }(rf.commitTrigger)
			// 在接收到快照之前，必须尽快通知上层服务，
			// 在任何新的提交信号之前，
			// 因此将commitTrigger设置为nil以阻止其他goroutine向该通道发送消息。
			// 一旦它开始处理快照并发送回上层服务，提交者将重新启用此通道
			rf.commitTrigger = nil
		}
	}()

	for i := range rf.Log {
		if rf.Log[i].Index == args.LastIncludedIndex && rf.Log[i].Term == args.LastIncludedTerm {
			// 如果现有的日志条目具有与快照的最后一个包含条目相同的索引和任期，
			// 保留其后的日志条目并返回
			rf.Log = rf.Log[i+1:]
			lablog.Debug(rf.me, lablog.Snap, "保留索引：%d 任期：%d之后的日志，剩余%d个日志", args.LastIncludedIndex, args.LastIncludedTerm, len(rf.Log))
			return
		}
	}

	// 丢弃整个日志
	rf.Log = make([]LogEntry, 0)
}

/********************** InstallSnapshot RPC调用者 ****************************/

// 构造InstallSnapshot RPC的参数，在持有互斥锁的情况下
func (rf *Raft) constructInstallSnapshotArgs() *InstallSnapshotArgs {
	return &InstallSnapshotArgs{
		Term:              rf.CurrentTerm,
		LeaderId:          rf.me,
		LastIncludedIndex: rf.LastIncludedIndex,
		LastIncludedTerm:  rf.LastIncludedTerm,
		Offset:            0,
		Data:              rf.persister.ReadSnapshot(),
		Done:              true,
	}
}

// 向服务器发送InstallSnapshot RPC
func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	lablog.Debug(rf.me, lablog.Snap, "T%d -> S%d 发送IS，(LII:%d LIT:%d)", args.Term, server, args.LastIncludedIndex, args.LastIncludedTerm)
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

// installSnapshot是单个对等体发送InstallSnapshot RPC并处理回复的goroutine
func (rf *Raft) installSnapshot(server int, term int, args *InstallSnapshotArgs, serialNo int) {
	reply := &InstallSnapshotReply{}
	r := rf.sendInstallSnapshot(server, args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 重新检查所有相关的假设
	// - 仍然是领导者
	// - 实例未被终止
	if rf.state != Leader || rf.killed() {
		return
	}

	if !r {
		// 服务器没有回复时重试
		select {
		case rf.installSnapshotCh[server] <- serialNo: // 重试使用serialNo
			lablog.Debug(rf.me, lablog.Drop, "-> S%d IS已丢弃：{T:%d LII:%d LIT:%d}，重试", server, args.Term, args.LastIncludedIndex, args.LastIncludedTerm)
		default:
		}
		return
	}

	if reply.Term > rf.CurrentTerm {
		// 如果RPC响应包含大于CurrentTerm的任期T：设置CurrentTerm = T，转变为跟随者
		lablog.Debug(rf.me, lablog.Term, "IS <- S%d 任期更高(%d > %d)，切换为跟随者", server, reply.Term, rf.CurrentTerm)
		rf.stepDown(reply.Term)
		rf.electionAlarm = nextElectionAlarm()
		return
	}

	// 重要：
	// 采用来自[students-guide-to-raft](https://thesquareplanet.com/blog/students-guide-to-raft/#term-confusion)“术语混淆”部分的建议
	// 引用：
	//  通过经验，我们发现最简单的方法是首先记录回复中的任期（可能比当前任期更高），
	//  然后将当前任期与您在原始RPC中发送的任期进行比较。
	//  如果两个任期不同，请丢弃回复并返回。
	//  只有在两个任期相同时，才应继续处理回复。
	if rf.CurrentTerm != term {
		return
	}

	// 如果成功，nextIndex应该只增加
	oldNextIndex := rf.nextIndex[server]
	// 如果网络重新排列了RPC回复，并且要更新的nextIndex <对等体的当前nextIndex，
	// 不要将nextIndex更新为较小的值（以避免重新发送对等体已经具有的日志条目）
	// matchIndex也是如此
	rf.nextIndex[server] = labutil.Max(args.LastIncludedIndex+1, rf.nextIndex[server])
	rf.matchIndex[server] = labutil.Max(args.LastIncludedIndex, rf.matchIndex[server])
	lablog.Debug(rf.me, lablog.Snap, "IS RPC -> S%d 成功，更新NI：%v，MI：%v", server, rf.nextIndex, rf.matchIndex)

	if rf.nextIndex[server] > oldNextIndex {
		// nextIndex已更新，告知snapshoter检查是否可以处理任何延迟的快照命令
		select {
		case rf.snapshotTrigger <- true:
		default:
		}
	}
}

// snapshotInstaller是一个goroutine，负责向服务器调用InstallSnapshot RPC，
// 在赢得选举后，领导者为其任期中的每个跟随者启动此goroutine，
// 一旦下台或终止，此goroutine将终止
func (rf *Raft) snapshotInstaller(server int, ch <-chan int, term int) {
	var lastArgs *InstallSnapshotArgs // 记录上次RPC的参数
	i := 1                            // serialNo
	currentSnapshotSendCnt := 0       // 正在发送的InstallSnapshot RPC调用的数量
	for !rf.killed() {
		serialNo, ok := <-ch
		if !ok {
			return // 通道已关闭，应终止此goroutine
		}
		rf.mu.Lock()

		switch {
		case rf.state != Leader || rf.CurrentTerm != term || rf.killed():
			// 重新检查
			rf.mu.Unlock()
			return
		case lastArgs == nil || rf.LastIncludedIndex > lastArgs.LastIncludedIndex:
			lastArgs = rf.constructInstallSnapshotArgs()
			// 重置正在发送的InstallSnapshot RPC调用的数量
			currentSnapshotSendCnt = 1
			// 增加此snapshotInstaller的serialNo
			i++
		case serialNo >= i:
			// 重试，需要发送上次安装快照请求的参数
			// 增加正在发送的InstallSnapshot RPC调用的数量
			i++
		case rf.LastIncludedIndex == lastArgs.LastIncludedIndex && currentSnapshotSendCnt < 3:
			// send more to speed up, in case of network delay or drop
			// increment count of outstanding InstallSnapshot RPC calls
			currentSnapshotSendCnt++
		default:
			rf.mu.Unlock()
			continue
		}

		go rf.installSnapshot(server, term, lastArgs, i) // provide i as serialNo, to pass back if retry

		rf.mu.Unlock()
	}
}
