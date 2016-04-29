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

const TIME_STEP = 10
const NUM_SCHEDULES = 10
const NUM_MUTATIONS = 3

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
	schedules        []Schedule
	currentTime      int
	startTime        map[Invocation]int
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
		schedules:        make([]Schedule, NUM_SCHEDULES),
		startTime:        make(map[Invocation]int),
	}
}

func (sc *Scheduler) NextTick() {
	sc.currentTime += TIME_STEP
	n := len(sc.schedules[0])
	if n != 0 && (sc.currentTime)%(n*TIME_STEP) == 0 {
		newSchedules := make([]Schedule, 0)
		for _, schedule := range sc.schedules {
			schedule = clean(schedule)
			newSchedules = append(newSchedules, schedule)
			if len(schedule) != 0 {
				newSchedules = append(newSchedules, mutate(schedule))
			}
		}
		sc.schedules = newSchedules
		scores := make([]int64, 0)
		for _, schedule := range sc.schedules {
			scores = append(scores, sc.evaluateSchedule(schedule))
		}
		sort.Sort(scheduleSorter{sc.schedules, scores})
		sc.schedules = sc.schedules[:NUM_SCHEDULES]
	}
}

func clean(schedule Schedule) Schedule {
	newSchedule := make(Schedule, 0)
	for _, s := range schedule {
		if !s.isDone {
			newSchedule = append(newSchedule, s)
		}
	}
	return newSchedule
}

func mutate(schedule Schedule) Schedule {
	newSchedule := append(Schedule{}, schedule...)
	for mutation := 0; mutation < NUM_MUTATIONS; mutation++ {
		i := rand.Intn(len(schedule))
		j := rand.Intn(len(schedule))
		newSchedule[i], newSchedule[j] = newSchedule[j], newSchedule[i]
	}
	return newSchedule
}

type scheduleSorter struct {
	schedules []Schedule
	scores    []int64
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
	for i := range sc.schedules {
		sc.schedules[i] = append(sc.schedules[i], s)
	}
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
	schedule := sc.schedules[0]
	for i := 0; sc.freeInvokerCount > 0 && len(schedule) > 0; i++ {
		j := 0
		s := schedule[0]
		if i < len(schedule) {
			j = i
			s = schedule[i]
			if s.runningTests != 0 {
				continue
			}
		}
		if s.isDone {
			schedule = append(schedule[:j], schedule[j+1:]...)
			i--
			continue
		}
		invocations = append(invocations, sc.NextInvocation(s))
		sc.freeInvokerCount--
	}
	sc.schedules[0] = schedule
	return invocations
}

func (sc *Scheduler) NextInvocation(s *Solution) Invocation {
	s.runningTests++
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
	s.runningTests--
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

func (sc *Scheduler) evaluateSchedule(schedule []*Solution) int64 {
	score := int64(0)
	t := sc.currentTime
	for _, s := range schedule {
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
