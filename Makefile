.DEFAULT_GOAL := evaluate

.PHONY: evaluate
evaluate: bin/scheduler bin/interactor bin/crossrun
	bin/crossrun

bin/scheduler:
	go build -o bin/scheduler src/scheduler.go

bin/crossrun:
	go build -o bin/crossrun src/crossrun.go

bin/interactor:
	clang++ -w -O2 -o bin/interactor vendor/interactor.cpp

.PHONY: clean
clean:
	$(RM) -r bin
