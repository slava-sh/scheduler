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
	rand.Seed(24536)
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
	tick := 0
	for ; in.HasMore(); sc.NextTick() {
		tick++
		n := sc.schedule.Len()
		if 0 < n && n < 1000 && tick%10 == 0 {
			s0 := sc.schedule.Get(0).(*Solution)
			if sc.schedule.Len() > s0.problem.testCount {
				oldScore := sc.evaluateSchedule(sc.schedule)
				newSchedule := sc.schedule.Copy()
				for i := 0; i < 3; i++ {
					newSchedule.MoveToFront(rand.Intn(n))
				}
				newScore := sc.evaluateSchedule(newSchedule)
				if newScore < oldScore {
					sc.schedule = newSchedule
					debug(sc.currentTime, "schedule score is", oldScore, "improved to", newScore)
				}
			}
		}
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
		for _, r := range sc.ScheduleInvocations() {
			//debug("scheduling test", r.test, "for", r.solutionId)
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
	schedule         *TreapArray
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
	isDone       bool
	nextTest     int
	testsRun     int
	timeConsumed int
	startTime    int
}

type Invocation struct {
	solutionId int
	test       int
}

func NewScheduler(invokerCount int) *Scheduler {
	return &Scheduler{
		invokerCount:     invokerCount,
		freeInvokerCount: invokerCount,
		schedule:         NewTreapArray(),
		startTime:        make(map[Invocation]int),
	}
}

func (sc *Scheduler) NextTick() {
	sc.currentTime += TIME_STEP
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
		startTime: sc.currentTime,
	}
	sc.solutions = append(sc.solutions, s)
	sc.schedule.PushBack(s)
	//debug("new", s, "for problem", problemId)
}

func (sc *Scheduler) HandleResponse(solutionId, test int, verdict string) {
	s := sc.solutions[solutionId]
	time := sc.currentTime - sc.startTime[Invocation{solutionId, test}]
	sc.setVerdict(s, test, verdict != "OK", time)
	sc.freeInvokerCount++
}

func (sc *Scheduler) ScheduleInvocations() []Invocation {
	invocations := make([]Invocation, 0)
	for sc.freeInvokerCount > 0 && sc.schedule.Len() > 0 {
		s := sc.schedule.Get(0).(*Solution)
		if s.isDone {
			sc.schedule.Remove(0)
			continue
		}
		invocations = append(invocations, sc.NextInvocation(s))
		sc.freeInvokerCount--
	}
	return invocations
}

func (sc *Scheduler) NextInvocation(s *Solution) Invocation {
	invocation := Invocation{s.id, s.nextTest}
	sc.startTime[invocation] = sc.currentTime
	s.nextTest++
	if s.nextTest == s.problem.testCount {
		//debug(s, "is done (all tests scheduled)")
		sc.setDone(s)
	}
	return invocation
}

func (sc *Scheduler) setVerdict(s *Solution, test int, rejected bool, time int) {
	s.testsRun++
	s.timeConsumed += time
	if rejected && !s.isDone {
		//debug(s, "is done (rejected)")
		sc.setDone(s)
	}
}

func (sc *Scheduler) setDone(s *Solution) {
	s.isDone = true
}

func (sc *Scheduler) evaluateSchedule(schedule *TreapArray) int64 {
	score := int64(0)
	t := sc.currentTime
	for i := 0; i < schedule.Len(); i++ {
		s := schedule.Get(i).(*Solution)
		if s.isDone {
			continue
		}
		testingTime := s.problem.timeLimit * (s.problem.testCount - s.testsRun)
		t += testingTime
		sTime := int64(t-s.startTime) / TIME_STEP
		score += sTime * sTime * sTime
	}
	return score
}

func (s *Solution) String() string {
	return fmt.Sprint("solution ", s.id)
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

func (t *TreapArray) Copy() *TreapArray {
	return &TreapArray{t.root.copy()}
}

func (node *Node) copy() *Node {
	if node == nil {
		return nil
	}
	return &Node{
		value:    node.value,
		size:     node.size,
		priority: node.priority,
		left:     node.left.copy(),
		right:    node.right.copy(),
	}
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

func (t *TreapArray) Remove(index int) interface{} {
	left, middle, right := split3(t.root, index, index)
	if middle == nil {
		panic("TreapArray: index error")
	}
	result := middle.value
	t.root = merge2(left, right)
	return result
}

func (t *TreapArray) MoveToFront(index int) {
	left, middle, right := split3(t.root, index, index)
	if middle == nil {
		panic("TreapArray: index error")
	}
	t.root = merge3(middle, left, right)
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
