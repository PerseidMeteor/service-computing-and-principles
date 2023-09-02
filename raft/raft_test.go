package raft

import (
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
)

func TestCommit(t *testing.T) {
	defer leaktest.CheckTimeout(t, 100*time.Millisecond)()

	rt := NewRaftTest(t, 10)
	defer rt.Shutdown()

	// 检查当前的leader
	origId, _ := rt.CheckCurrentLeader()

	// 提交commit
	rt.CommitToServer(origId, 1)
	rt.CommitToServer(origId, 2)

	time.Sleep(time.Duration(250) * time.Millisecond)

	// 检查commit是否存在, 以及当前的server数量
	rt.CheckCommittedWithNServers(2, 10)

	// leader断开连接
	rt.DisconnectPeer(origId)
	time.Sleep(time.Duration(10) * time.Millisecond)

	// commit提交到原来的leader节点
	rt.CommitToServer(origId, 3)
	time.Sleep(time.Duration(250) * time.Millisecond)

	// 检查commit是否存在，此处应该不存在
	rt.CheckNotCommitted(3)

	// 检查当前的leader
	newLeaderId, _ := rt.CheckCurrentLeader()

	// 提交commit
	rt.CommitToServer(newLeaderId, 4)
	time.Sleep(time.Duration(250) * time.Millisecond)

	// 检查commit是否存在, 以及当前的server数量
	rt.CheckCommittedWithNServers(4, 9)

	// 原来断连的leader重新连接
	rt.ReconnectPeer(origId)
	time.Sleep(time.Duration(600) * time.Millisecond)

	// 检查当前的leader，此处不应该为原来的leader，而是新的leader
	nextID, _ := rt.CheckCurrentLeader()
	if nextID == origId {
		t.Errorf("got finalLeaderId==origLeaderId==%d, want them different", nextID)
	}

	// 3的commit不应该提交
	rt.CheckNotCommitted(3)

	// leader再次断连和重连
	rt.DisconnectPeer(nextID)
	time.Sleep(time.Duration(600) * time.Millisecond)
	rt.ReconnectPeer(nextID)
	time.Sleep(time.Duration(600) * time.Millisecond)
}

type RaftTest struct {
	mu sync.Mutex

	// cluster is a list of all the raft servers participating in a cluster.
	cluster []*Server
	storage []*KVStorage

	commitChans []chan CommitEntry

	// commits has a list of commits per server in cluster.
	commits [][]CommitEntry

	// connected has a bool per server in cluster, specifying whether this server
	// is connected to the network
	connected []bool

	// alive has a bool per server in cluster, specifying whether this server is
	// alive (if false, it's crashed and won't respond to RPCs).
	alive []bool

	// n is the number of servers in the cluster.
	n int
	t *testing.T
}

func NewRaftTest(t *testing.T, n int) *RaftTest {
	ns := make([]*Server, n)
	connected := make([]bool, n)
	alive := make([]bool, n)
	commitChans := make([]chan CommitEntry, n)
	commits := make([][]CommitEntry, n)
	ready := make(chan interface{})
	storage := make([]*KVStorage, n)

	// Create all Servers in this cluster, assign ids and peer ids.
	for i := 0; i < n; i++ {
		peerIds := make([]int, 0)
		for p := 0; p < n; p++ {
			if p != i {
				peerIds = append(peerIds, p)
			}
		}

		storage[i] = NewMapStorage()
		commitChans[i] = make(chan CommitEntry)
		ns[i] = NewServer(i, peerIds, storage[i], ready, commitChans[i])
		ns[i].Serve()
		alive[i] = true
	}

	// Connect all peers to each other.
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j {
				ns[i].ConnectToPeer(j, ns[j].GetListenAddr())
			}
		}
		connected[i] = true
	}
	close(ready)

	h := &RaftTest{
		cluster:     ns,
		storage:     storage,
		commitChans: commitChans,
		commits:     commits,
		connected:   connected,
		alive:       alive,
		n:           n,
		t:           t,
	}
	for i := 0; i < n; i++ {
		go h.gotCommits(i)
	}
	return h
}

// Shutdown shuts down all the servers in the harness and waits for them to
// stop running.
func (rt *RaftTest) Shutdown() {
	for i := 0; i < rt.n; i++ {
		rt.cluster[i].DisconnectAll()
		rt.connected[i] = false
	}
	for i := 0; i < rt.n; i++ {
		if rt.alive[i] {
			rt.alive[i] = false
			rt.cluster[i].Shutdown()
		}
	}
	for i := 0; i < rt.n; i++ {
		close(rt.commitChans[i])
	}
}

// DisconnectPeer disconnects a server from all other servers in the cluster.
func (rt *RaftTest) DisconnectPeer(id int) {
	log.Printf("Disconnect %d", id)
	rt.cluster[id].DisconnectAll()
	for j := 0; j < rt.n; j++ {
		if j != id {
			rt.cluster[j].DisconnectPeer(id)
		}
	}
	rt.connected[id] = false
}

// ReconnectPeer connects a server to all other servers in the cluster.
func (rt *RaftTest) ReconnectPeer(id int) {
	log.Printf("Reconnect %d", id)
	for j := 0; j < rt.n; j++ {
		if j != id && rt.alive[j] {
			if err := rt.cluster[id].ConnectToPeer(j, rt.cluster[j].GetListenAddr()); err != nil {
				rt.t.Fatal(err)
			}
			if err := rt.cluster[j].ConnectToPeer(id, rt.cluster[id].GetListenAddr()); err != nil {
				rt.t.Fatal(err)
			}
		}
	}
	rt.connected[id] = true
}

// CheckCurrentLeader checks that there is a single leader in the cluster and
// returns its id and term.
func (rt *RaftTest) CheckCurrentLeader() (int, int) {
	for r := 0; r < 8; r++ {
		leaderId := -1
		leaderTerm := -1
		for i := 0; i < rt.n; i++ {
			if rt.connected[i] {
				_, term, isLeader := rt.cluster[i].node.Report()
				if isLeader {
					if leaderId < 0 {
						leaderId = i
						leaderTerm = term
					} else {
						rt.t.Fatalf("both %d and %d think they're leaders", leaderId, i)
					}
				}
			}
		}
		if leaderId >= 0 {
			return leaderId, leaderTerm
		}
		time.Sleep(150 * time.Millisecond)
	}

	rt.t.Fatalf("leader not found")
	return -1, -1
}

// CheckCommits checks that the given command has been committed.
func (rt *RaftTest) CheckCommits(cmd int) (nc int, index int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Find the length of the commits slice for connected servers.
	commitsLen := -1
	for i := 0; i < rt.n; i++ {
		if rt.connected[i] {
			if commitsLen >= 0 {
				// If this was set already, expect the new length to be the same.
				if len(rt.commits[i]) != commitsLen {
					rt.t.Fatalf("commits[%d] = %d, commitsLen = %d", i, rt.commits[i], commitsLen)
				}
			} else {
				commitsLen = len(rt.commits[i])
			}
		}
	}

	for c := 0; c < commitsLen; c++ {
		cmdAtC := -1
		for i := 0; i < rt.n; i++ {
			if rt.connected[i] {
				cmdOfN := rt.commits[i][c].Command.(int)
				if cmdAtC >= 0 {
					if cmdOfN != cmdAtC {
						rt.t.Errorf("got %d, want %d at h.commits[%d][%d]", cmdOfN, cmdAtC, i, c)
					}
				} else {
					cmdAtC = cmdOfN
				}
			}
		}
		if cmdAtC == cmd {
			// Check consistency of Index.
			index := -1
			nc := 0
			for i := 0; i < rt.n; i++ {
				if rt.connected[i] {
					if index >= 0 && rt.commits[i][c].Index != index {
						rt.t.Errorf("got Index=%d, want %d at h.commits[%d][%d]", rt.commits[i][c].Index, index, i, c)
					} else {
						index = rt.commits[i][c].Index
					}
					nc++
				}
			}
			return nc, index
		}
	}

	// If there's no early return, we haven't found the command we were looking
	// for.
	rt.t.Errorf("cmd=%d not found in commits", cmd)
	return -1, -1
}

// CheckCommittedWithNServers verifies that cmd was committed by exactly n connected
// servers.
func (rt *RaftTest) CheckCommittedWithNServers(cmd int, n int) {
	nc, _ := rt.CheckCommits(cmd)
	if nc != n {
		rt.t.Errorf("CheckCommittedN got nc=%d, want %d", nc, n)
	}
}

// CheckNotCommitted verifies that no command equal to cmd has been committed
// by any of the active servers yet.
func (rt *RaftTest) CheckNotCommitted(cmd int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for i := 0; i < rt.n; i++ {
		if rt.connected[i] {
			for c := 0; c < len(rt.commits[i]); c++ {
				gotCmd := rt.commits[i][c].Command.(int)
				if gotCmd == cmd {
					rt.t.Errorf("found %d at commits[%d][%d], expected none", cmd, i, c)
				}
			}
		}
	}
}

// CommitToServer commits the command to serverId.
func (rt *RaftTest) CommitToServer(serverId int, cmd interface{}) bool {
	return rt.cluster[serverId].node.Submit(cmd)
}

func (rt *RaftTest) gotCommits(i int) {
	for c := range rt.commitChans[i] {
		rt.mu.Lock()
		log.Printf("Commits(%d) got %+v", i, c)
		rt.commits[i] = append(rt.commits[i], c)
		rt.mu.Unlock()
	}
}

func init() {
	// output time firstly
	log.SetPrefix("time: ")
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	seed := time.Now().UnixNano()
	rand.Seed(seed)
}
