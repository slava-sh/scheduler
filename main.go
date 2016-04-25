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
		for {
			solution := nextInt()
			test := nextInt()
			if solution == -1 && test == -1 {
				break
			}
			verdict := nextWord()
			s.HandleResponse(solution, test, verdict)
		}
		for _, r := range s.ScheduleGrading() {
			fmt.Fprintln(out, r.solutionId, r.testId)
		}
		fmt.Fprintln(out, -1, -1)
		out.Flush()
	}
}

type Scheduler struct {
	invokerCount int
	problems     []*Problem
	solutions    []*Solution
}

type Problem struct {
	memoryLimit int
	testCount   int
}

type Solution struct {
	problemId int
	verdicts  []Verdict
}

type Verdict int

const (
	UNKNOWN Verdict = iota
	ACCEPTED
	REJECTED
)

type GradingRequest struct {
	solutionId int
	testId     int
}

func NewScheduler(invokerCount int) *Scheduler {
	return &Scheduler{
		invokerCount: invokerCount,
	}
}

func (s *Scheduler) AddProblem(memoryLimit, testCount int) {
	debug("problem", len(s.problems),
		"has", testCount, "tests",
		"and ML", memoryLimit, "ms")
	s.problems = append(s.problems, &Problem{memoryLimit, testCount})
}

func (s *Scheduler) AddSolution(problemId int) {
	solutionId := len(s.solutions)
	debug("new solution", solutionId, "for problem", problemId)
	p := s.problems[problemId]
	solution := &Solution{
		problemId,
		make([]Verdict, p.testCount),
	}
	s.solutions = append(s.solutions, solution)
}

func (s *Scheduler) HandleResponse(solution, test int, verdict string) {
	debug("verdict for", solution, "test", test, "is", verdict)
}

func (s *Scheduler) ScheduleGrading() []GradingRequest {
	requests := make([]GradingRequest, 0)
	return requests
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
