# temporal-bench
Stress driver for Temporal Server.
See https://docs.google.com/presentation/d/1pZa7viicZ5Y08DGmLYwKh605jGKgGF0AtWeNHlvu_30/edit?usp=sharing for a high-level description on how this works.

## Starting Workflow Bench Test
The driver application reads the following environment variables to connect to Temporal Server:
```
NAMESPACE=default
FRONTEND_ADDRESS=127.0.0.1:7233
NUM_DECISION_POLLERS=10
```

You will need to run it (the benchtest driver), which also acts as a Temporal worker. Use the makefile to do so:
```bash
make run
```
Then you can proceed to run a bench workflow as described below

# Start Basic Test Using Input File
```
tctl wf start --tq temporal-bench --wt bench-driver --wid 1 --wtt 5 --et 259200 --if ./basic-bench-driver-request.json
```
