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

func (solution *Solution) Priority() Priority {
	var p Priority
	p.progress = solution.testsRun * 100 / solution.problem.testCount / 20
	p.robin = solution.robin
	return p
}

const TIME_STEP = 10

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
	for ; in.HasMore(); s.NextTick() {
		for {
			problem := in.NextInt()
			if problem == -1 {
				break
			}
			s.AddSolution(problem)
		}
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
		debug("about", s.pendingSolutions.Len(), "solutions pending")
		for _, r := range s.ScheduleInvocations() {
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

func (s *Scheduler) NextTick() {
	s.currentTime += TIME_STEP
	debug("time is", s.currentTime)
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
	p := s.problems[problemId]
	solution := &Solution{
		id:        solutionId,
		problem:   p,
		verdicts:  make([]Verdict, p.testCount),
		startTime: s.currentTime,
		robin:     s.nextRobin(),
	}
	s.solutions = append(s.solutions, solution)
	s.pendingSolutions.Push(solution)
	debug("new", solution, "for problem", problemId)
}

func (s *Scheduler) HandleResponse(solutionId, test int, verdict string) {
	solution := s.solutions[solutionId]
	time := s.currentTime - s.startTime[Invocation{solutionId, test}]
	if verdict == "OK" {
		s.setVerdict(solution, test, ACCEPTED, time)
	} else {
		s.setVerdict(solution, test, REJECTED, time)
	}
	s.freeInvokerCount++
	debug("verdict for", solution, "test", test, "is", verdict, "took", time, "ms")
}

func (s *Scheduler) ScheduleInvocations() []Invocation {
	invocations := make([]Invocation, 0)
	for s.freeInvokerCount > 0 && s.pendingSolutions.Len() > 0 {
		solution := s.pendingSolutions.Top()
		invocations = append(invocations, s.NextInvocation(solution))
		s.freeInvokerCount--
	}
	return invocations
}

func (s *Scheduler) NextInvocation(solution *Solution) Invocation {
	invocation := Invocation{solution.id, solution.nextTest}
	s.startTime[invocation] = s.currentTime
	solution.robin = s.nextRobin()
	solution.nextTest++
	if solution.nextTest == solution.problem.testCount {
		debug(solution, "is done (all tests scheduled)")
		s.setDone(solution)
	}
	if solution.heapIndex != -1 {
		s.pendingSolutions.Update(solution.heapIndex)
	}
	return invocation
}

func (s *Scheduler) setVerdict(solution *Solution, test int, verdict Verdict, time int) {
	solution.verdicts[test] = verdict
	solution.testsRun++
	solution.timeConsumed += time
	if verdict == REJECTED {
		debug(solution, "is done (rejected)")
		s.setDone(solution)
	}
	if verdict == REJECTED || test == solution.problem.testCount-1 {
		debug(solution, "is done; time:",
			s.currentTime-solution.startTime, "total,",
			solution.timeConsumed, "consumed")
	}
	if solution.heapIndex != -1 {
		s.pendingSolutions.Update(solution.heapIndex)
	}
}

func (s *Scheduler) setDone(solution *Solution) {
	solution.isDone = true
	if solution.heapIndex != -1 {
		s.pendingSolutions.Remove(solution.heapIndex)
	}
}

func (s *Scheduler) nextRobin() int {
	s.robinGenereator++
	return s.robinGenereator
}

func (solution *Solution) String() string {
	return fmt.Sprint("solution ", solution.id)
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
