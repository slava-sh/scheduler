package main

import (
	"bufio"
	"bytes"
	"container/heap"
	"fmt"
	"io"
	"os"
	"strings"
)

type Priority struct {
	progress int
	robin    int
}

func (a Priority) Less(b Priority) bool {
	if a.progress != b.progress {
		return a.progress > b.progress
	}
	if a.robin != b.robin {
		return a.robin < b.robin
	}
	return false
}

func (s *Solution) Priority() Priority {
	var p Priority
	p.progress = s.testsRun * 100 / s.problem.testCount / 34
	p.robin = s.robin
	return p
}

const TIME_STEP = 10

func main() {
	in := NewFastReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	invokerCount := in.NextInt()
	problemCount := in.NextInt()
	sc := NewScheduler(invokerCount)
	for i := 0; i < problemCount; i++ {
		timeLimit := in.NextInt()
		testCount := in.NextInt()
		sc.AddProblem(timeLimit, testCount)
	}
	for ; in.HasMore(); sc.NextTick() {
		for {
			problem := in.NextInt()
			if problem == -1 {
				break
			}
			sc.AddSolution(problem)
		}
		for {
			s := in.NextInt()
			test := in.NextInt()
			if s == -1 && test == -1 {
				break
			}
			verdict := in.NextWord()
			sc.HandleResponse(s, test, verdict)
		}
		debug("free invokers:", sc.freeInvokerCount)
		debug("about", sc.pendingSolutions.Len(), "solutions pending")
		for _, r := range sc.ScheduleInvocations() {
			debug("scheduling test", r.test, "for", r.solutionId)
			fmt.Fprintln(out, r.solutionId, r.test)
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
	pendingSolutions *PriorityQueue
	currentTime      int
	startTime        map[Invocation]int
	robinGenereator  int
}

type Problem struct {
	id        int
	timeLimit int
	testCount int
}

type Solution struct {
	id           int
	problem      *Problem
	verdicts     []Verdict
	isDone       bool
	nextTest     int
	testsRun     int
	timeConsumed int
	startTime    int
	heapIndex    int
	robin        int
}

type Verdict int

const (
	UNKNOWN Verdict = iota
	ACCEPTED
	REJECTED
)

type Invocation struct {
	solutionId int
	test       int
}

func NewScheduler(invokerCount int) *Scheduler {
	return &Scheduler{
		invokerCount:     invokerCount,
		freeInvokerCount: invokerCount,
		pendingSolutions: NewPriorityQueue(),
		startTime:        make(map[Invocation]int),
	}
}

func (sc *Scheduler) NextTick() {
	sc.currentTime += TIME_STEP
	debug("time is", sc.currentTime)
}

func (sc *Scheduler) AddProblem(timeLimit, testCount int) {
	problemId := len(sc.problems)
	debug("problem", problemId,
		"has", testCount, "tests",
		"and ML", timeLimit, "ms")
	p := &Problem{problemId, timeLimit, testCount}
	sc.problems = append(sc.problems, p)
}

func (sc *Scheduler) AddSolution(problemId int) {
	solutionId := len(sc.solutions)
	p := sc.problems[problemId]
	s := &Solution{
		id:        solutionId,
		problem:   p,
		verdicts:  make([]Verdict, p.testCount),
		startTime: sc.currentTime,
		robin:     sc.nextRobin(),
	}
	sc.solutions = append(sc.solutions, s)
	sc.pendingSolutions.Push(s)
	debug("new", s, "for problem", problemId)
}

func (sc *Scheduler) HandleResponse(solutionId, test int, verdict string) {
	s := sc.solutions[solutionId]
	time := sc.currentTime - sc.startTime[Invocation{solutionId, test}]
	if verdict == "OK" {
		sc.setVerdict(s, test, ACCEPTED, time)
	} else {
		sc.setVerdict(s, test, REJECTED, time)
	}
	sc.freeInvokerCount++
	debug("verdict for", s, "test", test, "is", verdict, "took", time, "ms")
}

func (sc *Scheduler) ScheduleInvocations() []Invocation {
	invocations := make([]Invocation, 0)
	for sc.freeInvokerCount > 0 && sc.pendingSolutions.Len() > 0 {
		s := sc.pendingSolutions.Top()
		invocations = append(invocations, sc.NextInvocation(s))
		sc.freeInvokerCount--
	}
	return invocations
}

func (sc *Scheduler) NextInvocation(s *Solution) Invocation {
	invocation := Invocation{s.id, s.nextTest}
	sc.startTime[invocation] = sc.currentTime
	s.robin = sc.nextRobin()
	s.nextTest++
	if s.nextTest == s.problem.testCount {
		debug(s, "is done (all tests scheduled)")
		sc.setDone(s)
	}
	if s.heapIndex != -1 {
		sc.pendingSolutions.Update(s.heapIndex)
	}
	return invocation
}

func (sc *Scheduler) setVerdict(s *Solution, test int, verdict Verdict, time int) {
	s.verdicts[test] = verdict
	s.testsRun++
	s.timeConsumed += time
	if verdict == REJECTED {
		debug(s, "is done (rejected)")
		sc.setDone(s)
	}
	if verdict == REJECTED || test == s.problem.testCount-1 {
		debug(s, "is done; time:",
			sc.currentTime-s.startTime, "total,",
			s.timeConsumed, "consumed")
	}
	if s.heapIndex != -1 {
		sc.pendingSolutions.Update(s.heapIndex)
	}
}

func (sc *Scheduler) setDone(s *Solution) {
	s.isDone = true
	if s.heapIndex != -1 {
		sc.pendingSolutions.Remove(s.heapIndex)
	}
}

func (sc *Scheduler) nextRobin() int {
	sc.robinGenereator++
	return sc.robinGenereator
}

func (s *Solution) String() string {
	return fmt.Sprint("s ", s.id)
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

type PriorityQueue struct {
	heap pqHeap
}

func NewPriorityQueue() *PriorityQueue {
	pq := new(PriorityQueue)
	heap.Init(&pq.heap)
	return pq
}

func (pq *PriorityQueue) Push(item *Solution) {
	heap.Push(&pq.heap, item)
}

func (pq *PriorityQueue) Pop() *Solution {
	return heap.Pop(&pq.heap).(*Solution)
}

func (pq *PriorityQueue) Remove(index int) *Solution {
	return heap.Remove(&pq.heap, index).(*Solution)
}

func (pq *PriorityQueue) Update(index int) {
	heap.Fix(&pq.heap, index)
}

func (pq *PriorityQueue) Top() *Solution {
	return pq.heap[0]
}

func (pq *PriorityQueue) Len() int {
	return len(pq.heap)
}

type pqHeap []*Solution

func (h pqHeap) Less(i, j int) bool {
	return h[i].Priority().Less(h[j].Priority())
}

func (h pqHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}

func (h pqHeap) Len() int {
	return len(h)
}

func (h *pqHeap) Push(x interface{}) {
	item := x.(*Solution)
	item.heapIndex = len(*h)
	*h = append(*h, item)
}

func (h *pqHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.heapIndex = -1
	*h = old[:n-1]
	return item
}
