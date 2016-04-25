package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	defer out.Flush()
	invokerCount := nextInt()
	problemCount := nextInt()
	s := NewScheduler(invokerCount)
	for i := 0; i < problemCount; i++ {
		memoryLimit := nextInt()
		testCount := nextInt()
		s.AddProblem(memoryLimit, testCount)
	}
	for tick := 0; ; tick++ {
		debug("tick", tick)
		for {
			problem, isEOF := nextIntOrEOF()
			if isEOF {
				return
			}
			if problem == -1 {
				break
			}
			s.AddSolution(problem)
		}
		debug("free invokers:", s.freeInvokerCount)
		for {
			solution := nextInt()
			test := nextInt()
			if solution == -1 && test == -1 {
				break
			}
			verdict := nextWord()
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

type Scheduler struct {
	invokerCount     int
	freeInvokerCount int
	problems         []*Problem
	solutions        []*Solution
}

type Problem struct {
	id          int
	memoryLimit int
	testCount   int
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
	}
}

func (s *Scheduler) AddProblem(memoryLimit, testCount int) {
	problemId := len(s.problems)
	debug("problem", problemId,
		"has", testCount, "tests",
		"and ML", memoryLimit, "ms")
	p := &Problem{problemId, memoryLimit, testCount}
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
		solution := s.findWaitingSolution()
		if solution == nil {
			break
		}
		requests = append(requests, solution.RequestForNextTest())
		s.freeInvokerCount--
	}
	return requests
}

func (s *Scheduler) findWaitingSolution() *Solution {
	for _, solution := range s.solutions {
		if !solution.isDone {
			return solution
		}
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

var (
	in  = bufio.NewReader(os.Stdin)
	out = bufio.NewWriter(os.Stdout)
)

func nextIntOrEOF() (int, bool) {
	var x int
	_, err := fmt.Fscan(in, &x)
	return x, err == io.EOF
}

func nextInt() int {
	x, _ := nextIntOrEOF()
	return x
}

func nextWord() string {
	var w string
	fmt.Fscan(in, &w)
	return w
}

func debug(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}
