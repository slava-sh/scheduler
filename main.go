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
	for ; in.HasMore(); s.NextTick() {
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
		for _, r := range s.ScheduleInvokations() {
			debug("scheduling test", r.test, "for", r.solution)
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
	time             int
	startTime        map[Invokation]int
}

type Problem struct {
	id        int
	timeLimit int
	testCount int
}

type Solution struct {
	id          int
	problem     *Problem
	verdicts    []Verdict
	isDone      bool
	nextTest    int
	testsRun    int
	runningTime int
	e           *list.Element
}

type Verdict int

const (
	UNKNOWN Verdict = iota
	ACCEPTED
	REJECTED
)

type Invokation struct {
	solution *Solution
	test     int
}

func NewScheduler(invokerCount int) *Scheduler {
	return &Scheduler{
		invokerCount:     invokerCount,
		freeInvokerCount: invokerCount,
		pendingSolutions: list.New(),
		startTime:        make(map[Invokation]int),
	}
}

func (s *Scheduler) NextTick() {
	s.time += 10
	debug("time is", s.time)
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
		id:       solutionId,
		problem:  p,
		verdicts: make([]Verdict, p.testCount),
	}
	s.solutions = append(s.solutions, solution)
	solution.e = s.pendingSolutions.PushFront(solution)
	debug("new", solution, "for problem", problemId)
}

func (s *Scheduler) HandleResponse(solutionId, test int, verdict string) {
	solution := s.solutions[solutionId]
	time := s.time - s.startTime[Invokation{solution, test}]
	if verdict == "OK" {
		s.setVerdict(solution, test, ACCEPTED, time)
	} else {
		s.setVerdict(solution, test, REJECTED, time)
	}
	s.freeInvokerCount++
	debug("verdict for", solution, "test", test, "is", verdict, "took", time, "ms")
}

func (s *Scheduler) ScheduleInvokations() []Invokation {
	invokations := make([]Invokation, 0)
	for s.freeInvokerCount > 0 && s.pendingSolutions.Len() > 0 {
		e := s.pendingSolutions.Front()
		s.pendingSolutions.MoveToBack(e)
		solution := e.Value.(*Solution)
		invokations = append(invokations, s.NextInvokation(solution))
		s.freeInvokerCount--
	}
	return invokations
}

func (s *Scheduler) NextInvokation(solution *Solution) Invokation {
	invokation := Invokation{solution, solution.nextTest}
	solution.nextTest++
	if solution.nextTest == solution.problem.testCount {
		debug(solution, "is done (all tests)")
		s.setDone(solution)
	}
	return invokation
}

func (s *Scheduler) setVerdict(solution *Solution, test int, verdict Verdict, time int) {
	solution.verdicts[test] = verdict
	solution.testsRun++
	solution.runningTime += time
	if verdict == REJECTED {
		debug(solution, "is done (rejected)")
		s.setDone(solution)
	}
}

func (s *Scheduler) setDone(solution *Solution) {
	solution.isDone = true
	if solution.e != nil {
		s.pendingSolutions.Remove(solution.e)
		solution.e = nil
	}
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
