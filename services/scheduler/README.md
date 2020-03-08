Scheduler Service
=================

This describes how the complicated Scheduler service works.

For simplicity, users of this service need only call the `scheduler.NewScheduler()` 
function. This creates a `Scheduler` instance which manages everything.

The `Scheduler` instance exposes the following methods:

1. `Start` - this starts the `Scheduler` which means that a ticker will be run every
    5 seconds. At every ticker instance (5 seconds), the `Scheduler` will look for new
    jobs, fill its job queue and execute any outstanding jobs in the que
2. `Close` - this stops the `Scheduler` instance. This should be called to shutdown the
    `Scheduler` when the app closes
3. `AddJob` - this inserts a job to the top of the job queue

Triggers
========

There are 2 types of triggers, **Scheduled** and **Manual**.

1. **Scheduled** - this means that the job is scheduled via the Cron Expression 
    specified for the source 
2. **Manual** - this is used when a user manually inserts a job to the top of the
    job queue. Usually, this means that the job will be executed next.

Internals
=========

The primary classes within the scheduler package are listed below. They are not
needed and should not be used directly.

| Class | Description |
| :---- | :---------- |
| `JobManager` | Has 2 jobs. The first one looks for any job in the database and puts them in the `JobQueue`. The second one dispatches any jobs in the `JobQueue` |
| `JobQueue` | A thread-safe job queue, literally. It's FIFO. But has the option to put a job at the top of queue |
| `LogSlice` | A structure used to hold any logs |
| `TaskGroup` | A structure used to hold an entire task group. A task group is a single repo with all the steps and processes. It is made of one to many `StepGroup` which are run sequentially |
| `StepGroup` | A structure which holds a series of `Tasks`. `StepGroup`s are run sequentially while all tasks in the `StepGroup` are run in parallel |
| `Task` | A structure that defines a task that will be run with Docker |
