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
)

const (
	SEED                 = 24536
	GA_POPULATION        = 30
	GA_MATING_POPULATION = 5
	GA_MATING_CHILDREN   = 3
	GA_MUTATION_SWAPS    = 5
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
	updates := 0
	ticks := 0
	for in.HasMore() {
		ticks++
		for {
			problem := in.NextInt()
			if problem == -1 {
				break
			}
			sc.AddSolution(problem)
		}
		if ticks%(len(sc.schedule())*3+1) == 0 {
			sc.UpdateSchedules()
			updates++
		}
		for {
			solutionId := in.NextInt()
			test := in.NextInt()
			if solutionId == -1 && test == -1 {
				break
			}
			verdict := in.NextWord()
			sc.HandleResponse(solutionId, test, verdict)
		}
		for _, r := range sc.ScheduleInvocations() {
			fmt.Fprintln(out, r.solutionId, r.test)
		}
		fmt.Fprintln(out, -1, -1)
		out.Flush()
		sc.NextTick()
	}
	fmt.Fprintln(os.Stderr, updates, "updates in", ticks, "ticks")
	fmt.Fprintln(os.Stderr, len(sc.solutions), "solutions")
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
	if sc.freeInvokerCount == 0 {
		return nil
	}
	invocations := make([]Invocation, 0)
	for _, s := range sc.schedule() {
		if s.isDone || s.runningTests != 0 {
			continue
		}
		invocations = append(invocations, sc.nextInvocation(s))
		sc.freeInvokerCount--
		if sc.freeInvokerCount == 0 {
			break
		}
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
		t += sc.estimateInvokerTime(s)
		sTime := t - float64(s.startTime)
		score += sTime * sTime * sTime
	}
	return score
}

func (sc *Scheduler) estimateInvokerTime(s *Solution) float64 {
	if s.testsRun == 0 {
		return float64(s.problem.timeLimit * s.problem.testCount)
	} else {
		remainingRuns := s.problem.testCount - s.testsRun
		return float64(s.timeConsumed*remainingRuns) / float64(s.testsRun)
	}
}

func (sc *Scheduler) UpdateSchedules() {
	if len(sc.schedule()) == 0 {
		return
	}
	newSchedules := sc.generateNewSchedules()
	scores := make([]float64, 0)
	for _, schedule := range newSchedules {
		scores = append(scores, sc.scheduleScore(schedule.schedule))
	}
	sort.Sort(scheduleSorter{newSchedules, scores})

	sc.schedules = sc.schedules[:0]
	prevHash := int64(0)
	for i, schedule := range newSchedules {
		if len(sc.schedules) == GA_POPULATION {
			break
		}
		hash := schedule.schedule.hash()
		if i != 0 && hash == prevHash {
			continue
		}
		sc.schedules = append(sc.schedules, schedule.schedule)
		prevHash = hash

		if debugEnabled {
			line := new(bytes.Buffer)
			fmt.Fprintf(line, "%.0f %s %d:", scores[i], schedule.origin, sc.currentTime)
			for _, s := range schedule.schedule {
				fmt.Fprintf(line, " (%d, %d, %.0f),", s.id, s.startTime, sc.estimateInvokerTime(s))
			}
			debug(line.String())
		}
	}
	debug()
}

type NewSchedule struct {
	schedule Schedule
	origin   string
}

func (sc *Scheduler) generateNewSchedules() []NewSchedule {
	newSchedules := make([]NewSchedule, 0)
	for _, schedule := range sc.schedules {
		newSchedules = append(newSchedules, NewSchedule{clean(schedule), "clean"})
	}
	for i := 0; i < GA_POPULATION && i < len(newSchedules); i++ {
		newSchedules = append(newSchedules, NewSchedule{mutate(newSchedules[i].schedule), "mutate"})
		for j := i + 1; j < GA_MATING_POPULATION && j < len(newSchedules); j++ {
			for k := 0; k < GA_MATING_CHILDREN; k++ {
				newSchedules = append(newSchedules, NewSchedule{cross(newSchedules[i].schedule, newSchedules[j].schedule), "cross"})
			}
		}
	}
	return newSchedules
}

func (schedule Schedule) hash() int64 {
	result := int64(0)
	for _, s := range schedule {
		result = result*4999999 + int64(s.id)
	}
	return result
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
	if len(schedule) == 0 {
		return nil
	}
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
		if len(a) != 0 && rand.Intn(2) == 0 {
			s = a[0]
			a = a[1:]
		} else {
			s = b[0]
			b = b[1:]
		}
		if !used[s] {
			used[s] = true
			result = append(result, s)
		}
	}
	return result
}

type scheduleSorter struct {
	schedules []NewSchedule
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
