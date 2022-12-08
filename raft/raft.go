package raft

import (
	"errors"
)

var ErrInvalidRequestVoteBody = errors.New("invalid request vote body")

type Command struct {
	Key   []byte
	Value []byte
}

type Log struct {
	Index   uint64
	Term    uint64
	Command Command
}

type Server struct {
	logs                                                         []Log
	votedFor                                                     string
	currentTerm, commitIndex, lastApplied, nextIndex, matchIndex uint64
}

type RequestVote struct {
	Term, LastLogIndex, LastLogTerm uint64
}

type RequestVoteResult struct {
	Term        uint64
	VoteGranted bool
}

func (s *Server) OnRequestVote(rv *RequestVote) RequestVoteResult {
	if rv.Term < s.currentTerm {
		return RequestVoteResult{Term: s.currentTerm}
	}

	if s.votedFor == "" && rv.LastLogIndex >= uint64(len(s.logs)) {
		return RequestVoteResult{Term: s.currentTerm, VoteGranted: true}
	}

	return RequestVoteResult{Term: s.currentTerm}
}
