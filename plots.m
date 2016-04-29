function plots(test)
  data = csvread(sprintf('data/%02d.csv', test), 1, 0);
  submitTime  = data(:, 1);
  problem     = data(:, 2);
  invokerTime = data(:, 3);
  invocations = data(:, 4);
  testCount   = data(:, 5);

  subplot(2, 2, 1);
  scatter(submitTime, invokerTime, 8, problem, 'x');
  xlabel('submit time');
  ylabel('invoker time');

  subplot(2, 2, 2);
  scatter(submitTime, invocations, 8, problem, 'x');
  xlabel('submit time');
  ylabel('invocations');

  subplot(2, 2, 3);
  scatter(submitTime, testCount, 8, problem, 'x');
  xlabel('submit time');
  ylabel('test count');
end
