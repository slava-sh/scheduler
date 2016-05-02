package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

func run(test int) (score int, err error) {
	testFile := fmt.Sprintf("tests/%02d", test)
	outputFile, err := ioutil.TempFile("", "output")
	if err != nil {
		return
	}
	defer os.Remove(outputFile.Name())
	interactor := exec.Command("bin/interactor", testFile, outputFile.Name())
	scheduler := exec.Command("bin/scheduler")
	interactor.Stdin, err = scheduler.StdoutPipe()
	if err != nil {
		return
	}
	scheduler.Stdin, err = interactor.StdoutPipe()
	if err != nil {
		return
	}
	interactor.Stderr = os.Stderr
	scheduler.Stderr = os.Stderr
	interactor.Start()
	scheduler.Start()
	err = interactor.Wait()
	if err != nil {
		return
	}
	err = scheduler.Wait()
	if err != nil {
		return
	}
	output, err := bufio.NewReader(outputFile).ReadString('\n')
	if err != nil {
		return
	}
	return strconv.Atoi(output[:len(output)-1])
}

func main() {
	tests := make([]int, 0)
	if len(os.Args) > 1 {
		for _, word := range os.Args[1:] {
			test, err := strconv.Atoi(word)
			if err != nil {
				panic(err)
			}
			tests = append(tests, test)
		}
	} else {
		for i := 1; i <= 10; i++ {
			tests = append(tests, i)
		}
	}
	scores := make(map[int]int)
	fmt.Printf("Test\tScore\n")
	var wg sync.WaitGroup
	for _, test := range tests {
		wg.Add(1)
		go func(test int) {
			defer wg.Done()
			score, err := run(test)
			if err != nil {
				fmt.Printf("%v\t%v\n", test, err)
				return
			}
			scores[test] = score
			fmt.Printf("%v\t%v\n", test, score)
		}(test)
	}
	wg.Wait()
	sum := 0
	for _, score := range scores {
		sum += score
	}
	fmt.Printf("Score\t%v\n", sum)
}
