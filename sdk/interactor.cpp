#include "testlib.h"
#include <vector>
#include <cassert>

using namespace std;

struct Problem {
	int timeLimit, testCount;
	Problem(int timeLimit, int testCount) : 
			timeLimit(timeLimit), testCount(testCount) {}
	Problem() {}
};
vector<Problem> problems;

struct Verdict {
	int timeConsumed;
	bool passed;
};

struct Submission {
	int submitTime;
	int problemId;
	vector<Verdict> verdicts;
	vector<bool> judged;
	bool finished;
	int startTime, timeConsumed;
	Submission(int submitTime, int problemId, const vector<Verdict>& verdicts) :
			submitTime(submitTime), problemId(problemId), verdicts(verdicts) {
		const Problem& problem = problems[problemId];
		if (this->verdicts.empty()) {
			Verdict verdict;
			verdict.timeConsumed = 0;
			verdict.passed = false;
			this->verdicts.push_back(verdict);
		}
		while (this->verdicts.size() < problem.testCount)
			this->verdicts.push_back(this->verdicts.back());
		while (this->verdicts.size() > problem.testCount)
			this->verdicts.pop_back();
		judged.assign(problem.testCount, false);
		finished = false;
	}
};

vector<Submission> submissions;

set<pair<pair<int, bool>, pair<int, int> > > verdicts;
vector<int> invokerFreeTime;
int currentTime;
int finishedCount;

void checkFinished(Submission& submission) {
	if (submission.finished) return;
	for (int i = 0; i < int(submission.verdicts.size()); i++) {
		if (!submission.judged[i])
			return;
		if (!submission.verdicts[i].passed)
			break;
	}
	submission.timeConsumed = currentTime - submission.startTime;
	submission.finished = true;
	finishedCount++;
}

bool readOperations() {
	int submissionId, testId;
	while (scanf("%d%d", &submissionId, &testId) == 2) {
		if (submissionId == -1 && testId == -1)
			return true;
		if (submissionId < 0 || submissionId >= int(submissions.size()) || submissions[submissionId].submitTime > currentTime) {
			quitf(_wa, "Submission %d does not exist or is submitted after %d", submissionId, currentTime);
		}
		const Submission& submission = submissions[submissionId];
		if (testId < 0 || testId > int(submission.verdicts.size())) {
			quitf(_wa, "Test %d does not exist for problem %d", testId, submission.problemId);
		}
		bool foundInvoker = false;
		for (int i = 0; i < invokerFreeTime.size(); i++) {
		    int& freeTime = invokerFreeTime[i];
			if (freeTime <= currentTime) {
				foundInvoker = true;
				const Verdict& verdict = submission.verdicts[testId];
				freeTime = currentTime + verdict.timeConsumed;
				verdicts.insert(make_pair(make_pair(freeTime, verdict.passed), make_pair(submissionId, testId)));
				break;
			}
		}
		if (!foundInvoker) {
			// cerr << "Invoker not found" << endl;
		}
	}
	return false;
}

const int TIME_STEP = 10;

int main(int argc, char* argv[])
{
    setName("Scheduler");
    registerInteraction(argc, argv);
    
    int problemCount = inf.readInt();
    int invokerCount = inf.readInt();
    invokerFreeTime.clear();
    for (int i = 0; i < invokerCount; i++) {
    	invokerFreeTime.push_back(0);
    }
    
    //printf("%d\n", invokerCount);
    
    problems.resize(problemCount);
    for (int i = 0; i < problemCount; i++) {
    	int timeLimit = inf.readInt();
    	int testCount = inf.readInt();
    	if (testCount < 1) testCount = 1;
    	problems[i] = Problem(timeLimit, testCount);
    }
    
    //printf("%d\n", problemCount);
    for (int i = 0; i < problemCount; i++) {
    //	printf("%d %d\n", problems[i].timeLimit, problems[i].testCount);
    }
    //fflush(stdout);
    
    submissions.clear();
    printf("SubmitTime,Problem,InvokerTime,Invocations,TestCount,TL\n");
    while (true) {
    	int submitTime = inf.readInt();
        int invokerTime = 0;
        int invocations = 0;
    	if (submitTime == -1)
    		break;
    	int problemId = inf.readInt();
    	vector<Verdict> verdicts;
        bool isRJ = false;
    	while (true) {
    		int timeConsumed = inf.readInt();
    		if (timeConsumed == -1)
    			break;
    		string verdictString = inf.readToken();
    		Verdict verdict;
    		verdict.timeConsumed = timeConsumed;
    		verdict.passed = verdictString == "OK";
    		verdicts.push_back(verdict);
            if (!isRJ) {
                invokerTime += timeConsumed;
                invocations++;
            }
            if (!verdict.passed) {
                isRJ = true;
            }
    	}
    	if (!submissions.empty())
    	    assert(submissions.back().submitTime <= submitTime);
    	submissions.push_back(Submission(submitTime, problemId, verdicts));
        printf("%d,%d,%d,%d,%d,%d\n",
                submitTime, problemId, invokerTime,
                invocations, verdicts.size(), problems[problemId].timeLimit);
    }
    return 0;

   	finishedCount = 0; 
    int submissionIndex = 0;
    for (currentTime = 0; finishedCount != int(submissions.size()); currentTime += TIME_STEP) {
    	while (submissionIndex < int(submissions.size()) && submissions[submissionIndex].submitTime <= currentTime) {
			Submission& submission = submissions[submissionIndex];
			printf("%d\n", submission.problemId);
			submission.startTime = currentTime;
    		submissionIndex++;
    	}
    	puts("-1");
    	while (!verdicts.empty() && verdicts.begin()->first.first <= currentTime) {
    		submissions[verdicts.begin()->second.first].judged[verdicts.begin()->second.second] = true;
    		checkFinished(submissions[verdicts.begin()->second.first]);
    		printf("%d %d %s\n", verdicts.begin()->second.first, verdicts.begin()->second.second, 
    			verdicts.begin()->first.second ? "OK" : "RJ");
    		verdicts.erase(verdicts.begin());
    	}
    	puts("-1 -1");
    	fflush(stdout);
    	
    	if (!readOperations() && finishedCount != int(submissions.size())) {
			quitf(_wa, "Unexpected eof, not all submissions were judged");
    	}
    }
    
    const double k = 3;
    double sumK = 0;
    for (int i = 0; i < submissions.size(); i++) {
        const Submission& submission = submissions[i];
    	int judgeTime = submission.timeConsumed;
    	sumK += pow(judgeTime, k);
    }
    tout << int(pow(sumK / submissions.size(), 1 / k)) << endl;
    
    // just message
    quitf(_ok, "Finished in %d ms", currentTime);
}
