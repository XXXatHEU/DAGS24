package kvraft

import (
	"time"

	"6.824/labrpc"
	"6.824/labutil"
)

type Clerk struct {
	servers []*labrpc.ClientEnd // 服务器列表
	me      int64               // 客户端 ID
	leader  int                 // 记录上一次 RPC 中作为 leader 的服务器
	opId    int                 // 操作 ID，单调递增
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.me = labutil.Nrand()
	ck.leader = 0
	ck.opId = 1
	return ck
}

// 获取键值对中 key 对应的 value 值并返回
// 如果 key 不存在则返回 ""
// 在面对其他错误时会一直重试
func (ck *Clerk) Get(key string) string {
	args := &GetArgs{
		Key:      key,
		ClientId: ck.me,
		OpId:     ck.opId,
	}
	ck.opId++
	serverId := ck.leader // 首先将请求发送给 leader
	for {
		reply := &GetReply{}
		ok := ck.servers[serverId].Call("KVServer.Get", args, reply)
		if !ok || reply.Err == ErrWrongLeader || reply.Err == ErrShutdown {
			// 没有收到回复（可能是回复丢失、网络分区、服务器宕机等）或者收到了错误类型的回复，尝试发送给下一个服务器
			serverId = (serverId + 1) % len(ck.servers)
			continue
		}
		if reply.Err == ErrInitElection { // 如果刚刚发生了新一轮的选举，则等待一段时间后重试
			time.Sleep(100 * time.Millisecond)
			continue
		}

		ck.leader = serverId       // 记录当前 leader
		if reply.Err == ErrNoKey { // key 不存在时返回 ""
			return ""
		}
		if reply.Err == OK {
			return reply.Value // 找到 key 时返回 value
		}
	}
}

// Put 和 Append 共用的代码
func (ck *Clerk) PutAppend(key string, value string, op opType) {
	args := &PutAppendArgs{
		Key:      key,
		Value:    value,
		Op:       op,
		ClientId: ck.me,
		OpId:     ck.opId,
	}
	ck.opId++
	serverId := ck.leader // 首先将请求发送给 leader
	for {
		reply := &PutAppendReply{}
		ok := ck.servers[serverId].Call("KVServer.PutAppend", args, reply)
		if !ok || reply.Err == ErrWrongLeader || reply.Err == ErrShutdown {
			// 没有收到回复（可能是回复丢失、网络分区、服务器宕机等）或者收到了错误类型的回复，尝试发送给下一个服务器
			serverId = (serverId + 1) % len(ck.servers)
			continue
		}
		if reply.Err == ErrInitElection { // 如果刚刚发生了新一轮的选举，则等待一段时间后重试
			time.Sleep(100 * time.Millisecond)
			continue
		}

		ck.leader = serverId // 记录当前 leader
		if reply.Err == OK {
			return
		}
	}
}

// 将 key-value 对加入到分布式系统中
func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, opPut)
}

// 将 value 添加到 key 的原值后面
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, opAppend)
}
