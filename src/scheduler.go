package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
)

const TIME_STEP = 10

func main() {
	rand.Seed(98862)
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
	pendingSolutions *TreapArray
	currentTime      int
	startTime        map[Invocation]int
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
		pendingSolutions: NewTreapArray(),
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
	}
	s.solutions = append(s.solutions, solution)
	s.pendingSolutions.PushBack(solution)
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
		i := rand.Intn(s.pendingSolutions.Len())
		e := s.pendingSolutions.Get(i)
		solution := e.(*Solution)
		if solution.isDone {
			s.pendingSolutions.Remove(i)
			continue
		}
		invocations = append(invocations, s.NextInvocation(solution))
		s.freeInvokerCount--
	}
	return invocations
}

func (s *Scheduler) NextInvocation(solution *Solution) Invocation {
	invocation := Invocation{solution.id, solution.nextTest}
	s.startTime[invocation] = s.currentTime
	solution.nextTest++
	if solution.nextTest == solution.problem.testCount {
		debug(solution, "is done (all tests scheduled)")
		s.setDone(solution)
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
}

func (s *Scheduler) setDone(solution *Solution) {
	solution.isDone = true
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

type TreapArray struct {
	root *Node
}

type Node struct {
	value    interface{}
	size     int
	priority int
	left     *Node
	right    *Node
}

func NewNode(value interface{}) *Node {
	return &Node{
		value:    value,
		size:     1,
		priority: rand.Int(),
	}
}

func NewTreapArray() *TreapArray {
	return new(TreapArray)
}

func (t *TreapArray) Len() int {
	if t.root == nil {
		return 0
	}
	return t.root.size
}

func (t *TreapArray) PushBack(value interface{}) {
	t.root = merge2(t.root, NewNode(value))
}

func (t *TreapArray) Get(index int) interface{} {
	left, middle, right := split3(t.root, index, index)
	if middle == nil {
		panic("TreapArray: index error")
	}
	result := middle.value
	t.root = merge3(left, middle, right)
	return result
}

func (t *TreapArray) Remove(index int) {
	left, _, right := split3(t.root, index, index)
	t.root = merge2(left, right)
}

func split3(node *Node, a, b int) (left, middle, right *Node) {
	middle, right = split2(node, b)
	left, middle = split2(middle, a-1)
	return
}

func merge3(left, middle, right *Node) *Node {
	return merge2(merge2(left, middle), right)
}

func merge2(left, right *Node) *Node {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	var result *Node
	if left.priority > right.priority {
		result = left
		result.right = merge2(result.right, right)
	} else {
		result = right
		result.left = merge2(left, result.left)
	}
	update(result)
	return result
}

func split2(node *Node, index int) (left, right *Node) {
	if node == nil {
		return
	}
	nodeIndex := 0
	if node.left != nil {
		nodeIndex += node.left.size
	}
	if nodeIndex <= index {
		node.right, right = split2(node.right, index-nodeIndex-1)
		left = node
	} else {
		left, node.left = split2(node.left, index)
		right = node
	}
	update(node)
	return
}

func update(node *Node) {
	if node == nil {
		return
	}
	node.size = 1
	if node.left != nil {
		node.size += node.left.size
	}
	if node.right != nil {
		node.size += node.right.size
	}
}
