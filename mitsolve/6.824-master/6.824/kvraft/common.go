package kvraft

// 错误类型
type Err string

const (
	OK              = "OK"              // 操作成功
	ErrNoKey        = "ErrNoKey"        // 键不存在
	ErrWrongLeader  = "ErrWrongLeader"  // 请求发送到了非 leader 服务器
	ErrShutdown     = "ErrShutdown"     // 服务器已经关闭
	ErrInitElection = "ErrInitElection" // 刚刚发生了新一轮的选举
)

// 操作类型
type opType string

const (
	opGet    opType = "G" // 获取操作
	opPut    opType = "P" // 修改或添加操作
	opAppend opType = "A" // 追加操作
)

// Put 和 Append 共用的操作参数
type PutAppendArgs struct {
	Key      string // 键名
	Value    string // 键值
	Op       opType // 操作类型，默认为 Put
	ClientId int64  // 客户端 ID
	OpId     int    // 客户端操作 ID，单调递增
}

// Put 和 Append 共用的操作回复
type PutAppendReply struct {
	Err Err // 错误类型，可能为 OK 或 ErrWrongLeader、ErrShutdown
}

// Get 操作参数
type GetArgs struct {
	Key      string // 键名
	ClientId int64  // 客户端 ID
	OpId     int    // 客户端操作 ID，单调递增
}

// Get 操作回复
type GetReply struct {
	Err   Err    // 错误类型，可能为 OK、ErrNoKey、ErrWrongLeader、ErrShutdown
	Value string // 键对应的值，如果键不存在则为空字符串
}
