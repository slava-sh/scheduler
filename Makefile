.PHONY: debug
debug:
	go run -ldflags '-X main.debugFlag=true' scheduler.go

.PHONY: clean
clean:
	rm scheduler interactor output.txt

scheduler:
	go build scheduler.go

interactor:
	clang++ -O2 sdk/interactor.cpp -o interactor

crossrun: scheduler interactor
	java -jar sdk/CrossRun.jar "./interactor tests/$(TEST) output.txt" "./scheduler"
	@cat output.txt
