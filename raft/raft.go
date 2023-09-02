package raft

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

const DebugCM = 1

// CommitEntry is the data reported by Raft to the commit channel. Each commit
// entry notifies the client that consensus was reached on a command and it can
// be applied to the client's state machine.
type CommitEntry struct {
	// Command is the client command being committed.
	Command interface{}

	// Index is the log index at which the client command is committed.
	Index int

	// Term is the Raft term at which the client command is committed.
	Term int
}

// State is the state of a CM.
type State int

const (
	Follower State = iota
	Candidate
	Leader
	Dead
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Dead:
		return "Dead"
	default:
		panic("unreachable")
	}
}

type LogEntry struct {
	Command interface{}
	Term    int
}

type Node struct {
	mu sync.Mutex

	id int

	peerIds []int

	server *Server

	// 模拟节点的存储
	storage Storage

	// commitOKChan is an internal notification channel used by goroutines
	// that commit new entries to the log to notify that these entries may be sent
	// on commitChan.
	commitOKChan chan struct{}

	// commitChan is the channel where this CM is going to report committed log
	// entries. It's passed in by the client during construction.
	commitChan chan<- CommitEntry

	aEChan chan struct{}

	// Persistent Raft state on all servers
	// Term号
	currentTerm int
	// 是否投票
	votedFor int
	// log
	log []LogEntry

	// Volatile Raft state on all servers
	commitIndex        int
	lastApplied        int
	state              State
	electionResetEvent time.Time

	// Volatile Raft state on leaders
	nextIndex  map[int]int
	matchIndex map[int]int
}

func NewNode(id int, peerIds []int, server *Server, storage Storage, ready <-chan interface{}, commitChan chan<- CommitEntry) *Node {
	cm := new(Node)
	cm.id = id
	cm.peerIds = peerIds
	cm.server = server
	cm.storage = storage
	cm.commitChan = commitChan
	cm.commitOKChan = make(chan struct{}, 16)
	cm.aEChan = make(chan struct{}, 1)
	cm.state = Follower
	cm.votedFor = -1
	cm.commitIndex = -1
	cm.lastApplied = -1
	cm.nextIndex = make(map[int]int)
	cm.matchIndex = make(map[int]int)

	if cm.storage.StorageExists() {
		cm.restoreFromStorage()
	}

	go func() {
		// The CM is dormant until ready is signaled; then, it starts a countdown
		// for leader election.
		<-ready
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()
		cm.runElectionTimer()
	}()

	go cm.commitChanSender()
	return cm
}

// Report reports the state of this CM.
func (cm *Node) Report() (id int, term int, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.id, cm.currentTerm, cm.state == Leader
}

// Submit submits a new command to the raft node
func (node *Node) Submit(command interface{}) bool {
	node.mu.Lock()
	node.nodeLog("Submit received by %v: %v", node.state, command)
	if node.state == Leader {
		node.log = append(node.log, LogEntry{Command: command, Term: node.currentTerm})
		node.persistToStorage()
		node.nodeLog("... log=%v", node.log)
		node.mu.Unlock()
		node.aEChan <- struct{}{}
		return true
	}

	node.mu.Unlock()
	return false
}

func (cm *Node) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.state = Dead
	cm.nodeLog("becomes Dead")
	close(cm.commitOKChan)
}

func (cm *Node) restoreFromStorage() {
	if termData, found := cm.storage.Get("currentTerm"); found {
		d := gob.NewDecoder(bytes.NewBuffer(termData))
		if err := d.Decode(&cm.currentTerm); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("currentTerm not found in storage")
	}
	if votedData, found := cm.storage.Get("votedFor"); found {
		d := gob.NewDecoder(bytes.NewBuffer(votedData))
		if err := d.Decode(&cm.votedFor); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("votedFor not found in storage")
	}
	if logData, found := cm.storage.Get("log"); found {
		d := gob.NewDecoder(bytes.NewBuffer(logData))
		if err := d.Decode(&cm.log); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("log not found in storage")
	}
}

func (cm *Node) persistToStorage() {
	var termData bytes.Buffer
	if err := gob.NewEncoder(&termData).Encode(cm.currentTerm); err != nil {
		log.Fatal(err)
	}
	cm.storage.Set("currentTerm", termData.Bytes())

	var votedData bytes.Buffer
	if err := gob.NewEncoder(&votedData).Encode(cm.votedFor); err != nil {
		log.Fatal(err)
	}
	cm.storage.Set("votedFor", votedData.Bytes())

	var logData bytes.Buffer
	if err := gob.NewEncoder(&logData).Encode(cm.log); err != nil {
		log.Fatal(err)
	}
	cm.storage.Set("log", logData.Bytes())
}

// log to logfile with id prefix.
func (node *Node) nodeLog(format string, args ...interface{}) {
	format = fmt.Sprintf("server [%d] ", node.id) + format
	log.Printf(format, args...)
}

// RequestVote RPC to other node.
func (node *Node) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	node.mu.Lock()
	defer node.mu.Unlock()
	if node.state == Dead {
		return nil
	}
	lastLogIndex, lastLogTerm := node.lastLogIndexAndTerm()
	node.nodeLog("RequestVote: %+v [currentTerm=%d, votedFor=%d, log index/term=(%d, %d)]", args, node.currentTerm, node.votedFor, lastLogIndex, lastLogTerm)

	if args.Term > node.currentTerm {
		node.nodeLog("term out of date in RequestVote")
		node.becomeFollower(args.Term)
	}

	if node.currentTerm == args.Term &&
		(node.votedFor == -1 || node.votedFor == args.CandidateId) &&
		(args.LastLogTerm > lastLogTerm ||
			(args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex)) {
		reply.VoteGranted = true
		node.votedFor = args.CandidateId
		node.electionResetEvent = time.Now()
	} else {
		reply.VoteGranted = false
	}
	reply.Term = node.currentTerm
	node.persistToStorage()
	node.nodeLog("... RequestVote reply: %+v", reply)
	return nil
}

// AppendEntries RPC to other node.
func (node *Node) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	node.mu.Lock()
	defer node.mu.Unlock()
	if node.state == Dead {
		return nil
	}
	node.nodeLog("AppendEntries: %+v", args)

	if args.Term > node.currentTerm {
		node.nodeLog("... term out of date in AppendEntries")
		node.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == node.currentTerm {
		if node.state != Follower {
			node.becomeFollower(args.Term)
		}
		node.electionResetEvent = time.Now()

		if args.PrevLogIndex == -1 ||
			(args.PrevLogIndex < len(node.log) && args.PrevLogTerm == node.log[args.PrevLogIndex].Term) {
			reply.Success = true

			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0

			for {
				if logInsertIndex >= len(node.log) || newEntriesIndex >= len(args.Entries) {
					break
				}
				if node.log[logInsertIndex].Term != args.Entries[newEntriesIndex].Term {
					break
				}
				logInsertIndex++
				newEntriesIndex++
			}

			if newEntriesIndex < len(args.Entries) {
				node.nodeLog("inserting entries %v from index %d", args.Entries[newEntriesIndex:], logInsertIndex)
				node.log = append(node.log[:logInsertIndex], args.Entries[newEntriesIndex:]...)
				node.nodeLog("log is now: %v", node.log)
			}

			if args.LeaderCommit > node.commitIndex {
				node.commitIndex = Min(args.LeaderCommit, len(node.log)-1)
				node.nodeLog("setting commitIndex=%d", node.commitIndex)
				node.commitOKChan <- struct{}{}
			}
		} else {
			if args.PrevLogIndex >= len(node.log) {
				reply.ConflictIndex = len(node.log)
				reply.ConflictTerm = -1
			} else {
				reply.ConflictTerm = node.log[args.PrevLogIndex].Term

				var i int
				for i = args.PrevLogIndex - 1; i >= 0; i-- {
					if node.log[i].Term != reply.ConflictTerm {
						break
					}
				}
				reply.ConflictIndex = i + 1
			}
		}
	}

	reply.Term = node.currentTerm
	node.persistToStorage()
	node.nodeLog("AppendEntries reply: %+v", *reply)
	return nil
}

// electionTimeout generates a pseudo-random election timeout duration.
func (node *Node) electionTimeout() time.Duration {
	// If RAFT_FORCE_MORE_REELECTION is set, stress-test by deliberately
	// generating a hard-coded number very often. This will create collisions
	// between different servers and force more re-elections.
	if len(os.Getenv("RAFT_FORCE_MORE_REELECTION")) > 0 && rand.Intn(3) == 0 {
		return time.Duration(150) * time.Millisecond
	} else {
		return time.Duration(150+rand.Intn(150)) * time.Millisecond
	}
}

func (node *Node) runElectionTimer() {
	timeoutDuration := node.electionTimeout()
	node.mu.Lock()
	termStarted := node.currentTerm
	node.mu.Unlock()
	node.nodeLog("election timer started (%v), current term is %d", timeoutDuration, termStarted)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		node.mu.Lock()
		if node.state != Candidate && node.state != Follower {
			node.nodeLog("in election timer state=%s, bailing out", node.state)
			node.mu.Unlock()
			return
		}

		if termStarted != node.currentTerm {
			node.nodeLog("in election timer term changed from %d to %d, bailing out", termStarted, node.currentTerm)
			node.mu.Unlock()
			return
		}

		if elapsed := time.Since(node.electionResetEvent); elapsed >= timeoutDuration {
			node.startElection()
			node.mu.Unlock()
			return
		}
		node.mu.Unlock()
	}
}

func (node *Node) startElection() {
	node.state = Candidate
	node.currentTerm += 1
	savedCurrentTerm := node.currentTerm
	node.electionResetEvent = time.Now()
	node.votedFor = node.id
	node.nodeLog("becomes Candidate, current term is %d, current log is %v", savedCurrentTerm, node.log)

	votesReceived := 1

	for _, peerId := range node.peerIds {
		go func(peerId int) {
			node.mu.Lock()
			savedLastLogIndex, savedLastLogTerm := node.lastLogIndexAndTerm()
			node.mu.Unlock()

			args := RequestVoteArgs{
				Term:         savedCurrentTerm,
				CandidateId:  node.id,
				LastLogIndex: savedLastLogIndex,
				LastLogTerm:  savedLastLogTerm,
			}

			node.nodeLog("sending RequestVote to %d: %+v", peerId, args)
			var reply RequestVoteReply
			if err := node.server.Call(peerId, "ConsensusModule.RequestVote", args, &reply); err == nil {
				node.mu.Lock()
				defer node.mu.Unlock()
				node.nodeLog("received RequestVoteReply %+v", reply)

				if node.state != Candidate {
					node.nodeLog("while waiting for reply, state = %v", node.state)
					return
				}

				if reply.Term > savedCurrentTerm {
					node.nodeLog("term out of date in RequestVoteReply")
					node.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						if votesReceived*2 > len(node.peerIds)+1 {
							// Won the election!
							node.nodeLog("wins election with %d votes", votesReceived)
							node.startLeader()
							return
						}
					}
				}
			}
		}(peerId)
	}

	// Run another election timer, in case this election is not successful.
	go node.runElectionTimer()
}

// becomeFollower makes cm a follower and resets its state.
func (node *Node) becomeFollower(term int) {
	node.nodeLog("becomes Follower with term=%d; log=%v", term, node.log)
	node.state = Follower
	node.currentTerm = term
	node.votedFor = -1
	node.electionResetEvent = time.Now()

	go node.runElectionTimer()
}

// startLeader switches cm into a leader state and begins process of heartbeats.
func (node *Node) startLeader() {
	node.state = Leader

	for _, peerId := range node.peerIds {
		node.nextIndex[peerId] = len(node.log)
		node.matchIndex[peerId] = -1
	}
	node.nodeLog("becomes Leader; term=%d, nextIndex=%v, matchIndex=%v; log=%v", node.currentTerm, node.nextIndex, node.matchIndex, node.log)

	go func(heartbeatTimeout time.Duration) {
		// Immediately send AEs to peers.
		node.leaderSendAEs()

		t := time.NewTimer(heartbeatTimeout)
		defer t.Stop()
		for {
			doSend := false
			select {
			case <-t.C:
				doSend = true

				// Reset timer to fire again after heartbeatTimeout.
				t.Stop()
				t.Reset(heartbeatTimeout)
			case _, ok := <-node.aEChan:
				if ok {
					doSend = true
				} else {
					return
				}

				// Reset timer for heartbeatTimeout.
				if !t.Stop() {
					<-t.C
				}
				t.Reset(heartbeatTimeout)
			}

			if doSend {
				// If this isn't a leader any more, stop the heartbeat loop.
				node.mu.Lock()
				if node.state != Leader {
					node.mu.Unlock()
					return
				}
				node.mu.Unlock()
				node.leaderSendAEs()
			}
		}
	}(50 * time.Millisecond)
}

// leaderSendAEs sends a round of AEs to all peers, collects their
// replies and adjusts cm's state.
func (node *Node) leaderSendAEs() {
	node.mu.Lock()
	if node.state != Leader {
		node.mu.Unlock()
		return
	}
	savedCurrentTerm := node.currentTerm
	node.mu.Unlock()

	for _, peerId := range node.peerIds {
		go func(peerId int) {
			node.mu.Lock()
			ni := node.nextIndex[peerId]
			prevLogIndex := ni - 1
			prevLogTerm := -1
			if prevLogIndex >= 0 {
				prevLogTerm = node.log[prevLogIndex].Term
			}
			entries := node.log[ni:]

			args := AppendEntriesArgs{
				Term:         savedCurrentTerm,
				LeaderId:     node.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: node.commitIndex,
			}
			node.mu.Unlock()
			node.nodeLog("sending AppendEntries to %v: ni=%d, args=%+v", peerId, ni, args)
			var reply AppendEntriesReply
			if err := node.server.Call(peerId, "ConsensusModule.AppendEntries", args, &reply); err == nil {
				node.mu.Lock()
				defer node.mu.Unlock()
				if reply.Term > node.currentTerm {
					node.nodeLog("term out of date in heartbeat reply")
					node.becomeFollower(reply.Term)
					return
				}

				if node.state == Leader && savedCurrentTerm == reply.Term {
					if reply.Success {
						node.nextIndex[peerId] = ni + len(entries)
						node.matchIndex[peerId] = node.nextIndex[peerId] - 1

						savedCommitIndex := node.commitIndex
						for i := node.commitIndex + 1; i < len(node.log); i++ {
							if node.log[i].Term == node.currentTerm {
								matchCount := 1
								for _, peerId := range node.peerIds {
									if node.matchIndex[peerId] >= i {
										matchCount++
									}
								}
								if matchCount*2 > len(node.peerIds)+1 {
									node.commitIndex = i
								}
							}
						}
						node.nodeLog("AppendEntries reply from %d success: nextIndex := %v, matchIndex := %v; commitIndex := %d", peerId, node.nextIndex, node.matchIndex, node.commitIndex)
						if node.commitIndex != savedCommitIndex {
							node.nodeLog("leader sets commitIndex := %d", node.commitIndex)
							// Commit index changed: the leader considers new entries to be
							// committed. Send new entries on the commit channel to this
							// leader's clients, and notify followers by sending them AEs.
							node.commitOKChan <- struct{}{}
							node.aEChan <- struct{}{}
						}
					} else {
						if reply.ConflictTerm >= 0 {
							lastIndexOfTerm := -1
							for i := len(node.log) - 1; i >= 0; i-- {
								if node.log[i].Term == reply.ConflictTerm {
									lastIndexOfTerm = i
									break
								}
							}
							if lastIndexOfTerm >= 0 {
								node.nextIndex[peerId] = lastIndexOfTerm + 1
							} else {
								node.nextIndex[peerId] = reply.ConflictIndex
							}
						} else {
							node.nextIndex[peerId] = reply.ConflictIndex
						}
						node.nodeLog("AppendEntries reply from %d !success: nextIndex := %d", peerId, ni-1)
					}
				}
			}
		}(peerId)
	}
}

func (node *Node) lastLogIndexAndTerm() (int, int) {
	if len(node.log) > 0 {
		lastIndex := len(node.log) - 1
		return lastIndex, node.log[lastIndex].Term
	} else {
		return -1, -1
	}
}

func (node *Node) commitChanSender() {
	for range node.commitOKChan {
		// Find which entries we have to apply.
		node.mu.Lock()
		savedTerm := node.currentTerm
		savedLastApplied := node.lastApplied
		var entries []LogEntry
		if node.commitIndex > node.lastApplied {
			entries = node.log[node.lastApplied+1 : node.commitIndex+1]
			node.lastApplied = node.commitIndex
		}
		node.mu.Unlock()
		node.nodeLog("commitChanSender entries=%v, savedLastApplied=%d", entries, savedLastApplied)

		for i, entry := range entries {
			node.commitChan <- CommitEntry{
				Command: entry.Command,
				Index:   savedLastApplied + i + 1,
				Term:    savedTerm,
			}
		}
	}
	node.nodeLog("commitChanSender done")
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
