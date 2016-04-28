java -jar CrossRun.jar "./interactor tests/$1 output.txt" "$2"

if [ "$?" != "0" ]; then
	echo Some error occured
	exit 1
fi

echo Processes are finished, the calculated value is:
cat output.txt
