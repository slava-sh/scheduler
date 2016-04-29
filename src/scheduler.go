package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	SEED              = 24536
	GA_POPULATION     = 10
	GA_MUTATION_SWAPS = 5
	UPDATE_TIME       = 1 * time.Millisecond
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
	updateCount := 0
	go func() {
		for {
			sc.UpdateSchedules()
			updateCount++
			time.Sleep(UPDATE_TIME)
		}
	}()
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
	fmt.Fprintln(os.Stderr, updateCount, "updates")
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
	schedulesMutex   *sync.Mutex
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
		schedulesMutex:   new(sync.Mutex),
		invocationTime:   make(map[Invocation]int),
	}
}

func (sc *Scheduler) NextTick() {
	sc.currentTime += TIME_STEP
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
	sc.schedulesMutex.Lock()
	for i := range sc.schedules {
		sc.schedules[i] = append(sc.schedules[i], s)
	}
	sc.schedulesMutex.Unlock()
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
		for !s.isDone && sc.freeInvokerCount != 0 {
			invocations = append(invocations, sc.nextInvocation(s))
			sc.freeInvokerCount--
		}
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

func (sc *Scheduler) scheduleScore(schedule Schedule) float64 {
	score := float64(0)
	t := float64(sc.currentTime)
	for _, s := range schedule {
		if s.isDone {
			continue
		}
		var estTime float64
		if s.testsRun == 0 {
			estTime = float64(s.problem.timeLimit * s.problem.testCount)
		} else {
			remainingRuns := s.problem.testCount - s.testsRun
			estTime = float64(s.timeConsumed*remainingRuns) / float64(s.testsRun)
		}
		t += estTime
		sTime := (t - float64(s.startTime)) / TIME_STEP
		score += sTime * sTime * sTime
	}
	return score
}

func (sc *Scheduler) UpdateSchedules() {
	sc.schedulesMutex.Lock()
	newSchedules := make([]Schedule, 0)
	for _, schedule := range sc.schedules {
		newSchedules = append(newSchedules, clean(schedule))
	}
	if len(newSchedules[0]) != 0 {
		for i := 0; i < GA_POPULATION; i++ {
			for j := i + 1; j < GA_POPULATION; j++ {
				child := cross(newSchedules[i], newSchedules[j])
				newSchedules = append(newSchedules, child)
				if len(child) != 0 {
					newSchedules = append(newSchedules, mutate(child))
				}
			}
		}
	}
	sc.schedules = newSchedules
	scores := make([]float64, 0)
	for _, schedule := range sc.schedules {
		scores = append(scores, sc.scheduleScore(schedule))
	}
	sort.Sort(scheduleSorter{sc.schedules, scores})
	sc.schedules = sc.schedules[:GA_POPULATION]
	sc.schedulesMutex.Unlock()
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
	for mutation := 0; mutation < GA_MUTATION_SWAPS; mutation++ {
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
	scores    []float64
}

func (s scheduleSorter) Len() int {
	return len(s.schedules)
}

func (s scheduleSorter) Swap(i, j int) {
	s.schedules[i], s.schedules[j] = s.schedules[j], s.schedules[i]
	s.scores[i], s.scores[j] = s.scores[j], s.scores[i]
}

func (s scheduleSorter) Less(i, j int) bool {
	return s.scores[i] < s.scores[j]
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
