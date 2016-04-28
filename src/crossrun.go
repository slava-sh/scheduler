package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type logWriter log.Logger

func (w *logWriter) Write(b []byte) (int, error) {
	(*log.Logger)(w).Print(string(b))
	return len(b), nil
}

func spyWriter(w io.Writer, name string) io.Writer {
	lw := (*logWriter)(log.New(os.Stdout, fmt.Sprint(name, ": "), log.LstdFlags))
	return io.MultiWriter(w, lw)
}

func run(test int) int {
	testFile := fmt.Sprintf("tests/%02d", test)
	outputFile, err := ioutil.TempFile("", "output")
	if err != nil {
		log.Panic(err)
	}
	defer os.Remove(outputFile.Name())
	interactor := exec.Command("bin/interactor", testFile, outputFile.Name())
	scheduler := exec.Command("bin/scheduler")
	interactor.Stdin, err = scheduler.StdoutPipe()
	if err != nil {
		log.Panic(err)
	}
	scheduler.Stdin, err = interactor.StdoutPipe()
	if err != nil {
		log.Panic(err)
	}
	//interactor.Stderr = os.Stderr
	scheduler.Stderr = os.Stderr
	interactor.Start()
	scheduler.Start()
	err = interactor.Wait()
	if err != nil {
		log.Panic(err)
	}
	err = scheduler.Wait()
	if err != nil {
		log.Panic(err)
	}
	output, err := bufio.NewReader(outputFile).ReadString('\n')
	if err != nil {
		log.Panic(err)
	}
	score, err := strconv.Atoi(output[:len(output)-1])
	if err != nil {
		log.Panic(err)
	}
	return score
}

func main() {
	const NUM_TESTS = 10
	scores := make([]int, NUM_TESTS)
	var wg sync.WaitGroup
	fmt.Printf("Test\tScore\n")
	for i := range scores {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			score := run(i + 1)
			scores[i] = score
			fmt.Printf("%v\t%v\n", i+1, score)
		}(i)
	}
	wg.Wait()
	sum := 0
	for _, score := range scores {
		sum += score
	}
	fmt.Printf("Total\t%v\n", sum)
}
