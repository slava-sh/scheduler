package main

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	in := NewFastReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	invokerCount := in.NextInt()
	problemCount := in.NextInt()
	s := NewScheduler(invokerCount)
	for i := 0; i < problemCount; i++ {
		timeLimit := in.NextInt()
		testCount := in.NextInt()
		s.AddProblem(timeLimit, testCount)
	}
	for tick := 0; in.HasMore(); tick++ {
		debug("tick", tick)
		for {
			problem := in.NextInt()
			if problem == -1 {
				break
			}
			s.AddSolution(problem)
		}
		debug("free invokers:", s.freeInvokerCount)
		for {
			solution := in.NextInt()
			test := in.NextInt()
			if solution == -1 && test == -1 {
				break
			}
			verdict := in.NextWord()
			s.HandleResponse(solution, test, verdict)
		}
		debug("free invokers:", s.freeInvokerCount)
		for _, r := range s.ScheduleGrading() {
			debug("scheduling test", r.test, "for solution", r.solution.id)
			fmt.Fprintln(out, r.solution.id, r.test)
		}
		fmt.Fprintln(out, -1, -1)
		out.Flush()
	}
}

var debugFlag string
var debugEnabled = len(debugFlag) != 0

func debug(a ...interface{}) {
	if !debugEnabled {
		return
	}
	fmt.Fprintln(os.Stderr, a...)
}

type Scheduler struct {
	invokerCount     int
	freeInvokerCount int
	problems         []*Problem
	solutions        []*Solution
	pendingSolutions *list.List
}

type Problem struct {
	id        int
	timeLimit int
	testCount int
}

type Solution struct {
	id       int
	problem  *Problem
	verdicts []Verdict
	isDone   bool
	nextTest int
}

type Verdict int

const (
	UNKNOWN Verdict = iota
	ACCEPTED
	REJECTED
)

type GradingRequest struct {
	solution *Solution
	test     int
}

func NewScheduler(invokerCount int) *Scheduler {
	return &Scheduler{
		invokerCount:     invokerCount,
		freeInvokerCount: invokerCount,
		pendingSolutions: list.New(),
	}
}

func (s *Scheduler) AddProblem(timeLimit, testCount int) {
	problemId := len(s.problems)
	debug("problem", problemId,
		"has", testCount, "tests",
		"and ML", timeLimit, "ms")
	p := &Problem{problemId, timeLimit, testCount}
	s.problems = append(s.problems, p)
}

func (s *Scheduler) AddSolution(problemId int) {
	solutionId := len(s.solutions)
	debug("new solution", solutionId, "for problem", problemId)
	p := s.problems[problemId]
	solution := &Solution{
		id:       solutionId,
		problem:  p,
		verdicts: make([]Verdict, p.testCount),
	}
	s.solutions = append(s.solutions, solution)
	s.pendingSolutions.PushBack(solution)
}

func (s *Scheduler) HandleResponse(solutionId, test int, verdict string) {
	debug("verdict for", solutionId, "test", test, "is", verdict)
	s.freeInvokerCount++
	solution := s.solutions[solutionId]
	if verdict == "OK" {
		solution.SetVerdict(test, ACCEPTED)
	} else {
		solution.SetVerdict(test, REJECTED)
	}
}

func (s *Scheduler) ScheduleGrading() []GradingRequest {
	requests := make([]GradingRequest, 0)
	for s.freeInvokerCount > 0 {
		solution := s.findPendingSolution()
		if solution == nil {
			break
		}
		requests = append(requests, solution.RequestForNextTest())
		s.freeInvokerCount--
	}
	return requests
}

func (s *Scheduler) findPendingSolution() *Solution {
	for e := s.pendingSolutions.Front(); e != nil; {
		solution := e.Value.(*Solution)
		if !solution.isDone {
			return solution
		}
		next := e.Next()
		s.pendingSolutions.Remove(e)
		e = next
	}
	return nil
}

func (solution *Solution) RequestForNextTest() GradingRequest {
	request := GradingRequest{solution, solution.nextTest}
	solution.nextTest++
	if solution.nextTest == solution.problem.testCount {
		solution.isDone = true
	}
	return request
}

func (solution *Solution) SetVerdict(test int, verdict Verdict) {
	solution.verdicts[test] = verdict
	if verdict == REJECTED {
		solution.isDone = true
	}
}

type FastReader struct {
	r     *bufio.Reader
	words []string
}

func NewFastReader(r io.Reader) *FastReader {
	return &FastReader{
		r: bufio.NewReader(r),
	}
}

func (r *FastReader) advance() {
	if len(r.words) != 0 {
		return
	}
	var buf bytes.Buffer
	for {
		chunk, more, _ := r.r.ReadLine()
		buf.Write(chunk)
		if !more {
			break
		}
	}
	r.words = strings.FieldsFunc(buf.String(), func(c rune) bool {
		return c == ' '
	})
}

func (r *FastReader) HasMore() bool {
	r.advance()
	return len(r.words) != 0
}

func (r *FastReader) NextWord() string {
	r.advance()
	word := r.words[0]
	r.words = r.words[1:]
	return word
}

func (r *FastReader) NextInt() int {
	return parseInt(r.NextWord())
}

func parseInt(word string) int {
	sign := 1
	if word[0] == '-' {
		sign = -1
		word = word[1:]
	}
	result := 0
	for i := 0; i < len(word); i++ {
		result = result*10 + int(word[i]) - '0'
	}
	result *= sign
	return result
}
