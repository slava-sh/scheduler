.PHONY: debug
debug:
	go run -ldflags '-X main.debugFlag=true' main.go

.PHONY: run
run:
	go run main.go

.PHONY: clean
clean:
	rm main
