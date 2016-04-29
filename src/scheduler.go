package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"strings"
)

const (
	SEED          = 24536
	GA_POPULATION = 10
	GA_MUTATIONS  = 5
)

const TIME_STEP = 10

func main() {
	rand.Seed(SEED)
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
		for _, r := range sc.ScheduleInvocations() {
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
	schedules        []Schedule
	currentTime      int
	invocationTime   map[Invocation]int
}

type Schedule []*Solution

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
	runningTests int
}

type Invocation struct {
	solutionId int
	test       int
}

func NewScheduler(invokerCount int) *Scheduler {
	return &Scheduler{
		invokerCount:     invokerCount,
		freeInvokerCount: invokerCount,
		schedules:        make([]Schedule, GA_POPULATION),
		invocationTime:   make(map[Invocation]int),
	}
}

func (sc *Scheduler) NextTick() {
	sc.currentTime += TIME_STEP
	n := len(sc.schedule())
	if n != 0 && (sc.currentTime)%(TIME_STEP*2*n) == 0 {
		sc.updateSchedules()
	}
}

func (sc *Scheduler) AddProblem(timeLimit, testCount int) {
	problemId := len(sc.problems)
	debug("problem", problemId, "has", testCount, "tests and ML", timeLimit, "ms")
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
	for i := range sc.schedules {
		sc.schedules[i] = append(sc.schedules[i], s)
	}
}

func (sc *Scheduler) HandleResponse(solutionId, test int, verdict string) {
	time := sc.currentTime - sc.invocationTime[Invocation{solutionId, test}]
	s := sc.solutions[solutionId]
	s.runningTests--
	s.testsRun++
	s.timeConsumed += time
	if verdict != "OK" {
		s.isDone = true
	}
	sc.freeInvokerCount++
}

func (sc *Scheduler) ScheduleInvocations() []Invocation {
	invocations := make([]Invocation, 0)
	for _, s := range sc.schedule() {
		if sc.freeInvokerCount == 0 {
			break
		}
		if s.isDone || s.runningTests != 0 {
			continue
		}
		invocations = append(invocations, sc.nextInvocation(s))
		sc.freeInvokerCount--
	}
	for _, s := range sc.schedule() {
		if sc.freeInvokerCount == 0 {
			break
		}
		if s.isDone {
			continue
		}
		invocations = append(invocations, sc.nextInvocation(s))
		sc.freeInvokerCount--
	}
	return invocations
}

func (sc *Scheduler) nextInvocation(s *Solution) Invocation {
	s.runningTests++
	invocation := Invocation{s.id, s.nextTest}
	sc.invocationTime[invocation] = sc.currentTime
	s.nextTest++
	if s.nextTest == s.problem.testCount {
		s.isDone = true
	}
	return invocation
}

func (sc *Scheduler) schedule() Schedule {
	return sc.schedules[0]
}

func (sc *Scheduler) evaluateSchedule(schedule Schedule) *big.Int {
	score := big.NewInt(0)
	t := sc.currentTime
	for _, s := range schedule {
		if s.isDone {
			continue
		}
		testingTime := 0
		if s.testsRun == 0 {
			testingTime = s.problem.timeLimit * (s.problem.testCount - s.testsRun)
		} else {
			testingTime = s.timeConsumed * (s.problem.testCount - s.testsRun) / s.testsRun
		}
		t += testingTime
		sTime := big.NewInt(int64(t-s.startTime) / TIME_STEP)
		sTime3 := big.NewInt(0)
		sTime3.Mul(sTime, sTime)
		sTime3.Mul(sTime3, sTime)
		score.Add(score, sTime3)
	}
	return score
}

func (sc *Scheduler) updateSchedules() {
	newSchedules := make([]Schedule, 0)
	for _, schedule := range sc.schedules {
		schedule = clean(schedule)
		newSchedules = append(newSchedules, schedule)
		if len(schedule) != 0 {
			newSchedules = append(newSchedules, mutate(schedule))
		}
	}
	sc.schedules = newSchedules
	scores := make([]*big.Int, 0)
	for _, schedule := range sc.schedules {
		scores = append(scores, sc.evaluateSchedule(schedule))
	}
	sort.Sort(scheduleSorter{sc.schedules, scores})
	sc.schedules = sc.schedules[:GA_POPULATION]
}

func clean(schedule Schedule) Schedule {
	result := make(Schedule, 0)
	for _, s := range schedule {
		if !s.isDone {
			result = append(result, s)
		}
	}
	return result
}

func mutate(schedule Schedule) Schedule {
	result := append(Schedule{}, schedule...)
	for mutation := 0; mutation < GA_MUTATIONS; mutation++ {
		i := rand.Intn(len(schedule))
		j := rand.Intn(len(schedule))
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func cross(a, b Schedule) Schedule {
	result := make(Schedule, 0)
	used := make(map[*Solution]bool)
	for len(a) != 0 && len(b) != 0 {
		var s *Solution
		if len(a) == 0 && rand.Intn(1) == 0 {
			s = b[0]
			b = b[1:]
		} else {
			s = a[0]
			a = a[1:]
		}
		if !used[s] {
			used[s] = true
			result = append(result, s)
		}
	}
	return result
}

type scheduleSorter struct {
	schedules []Schedule
	scores    []*big.Int
}

func (s scheduleSorter) Len() int {
	return len(s.schedules)
}

func (s scheduleSorter) Swap(i, j int) {
	s.schedules[i], s.schedules[j] = s.schedules[j], s.schedules[i]
	s.scores[i], s.scores[j] = s.scores[j], s.scores[i]
}

func (s scheduleSorter) Less(i, j int) bool {
	return s.scores[i].Cmp(s.scores[j]) < 0
}

func (s *Solution) String() string {
	return fmt.Sprint(s.id)
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
