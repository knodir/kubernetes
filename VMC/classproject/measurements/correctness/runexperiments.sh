#!/bin/bash

# finished execution.
STRING="Finished"

# echoclient should finish within this interval. In seconds.
# Fixme change 3 to 15 in real experiment
TOTAL_EXPERIMENT_TIME=3

# send one packet with 1000 us interval. Stop after sending 10K packets. 
# Fixme change 10 to 10000 in real experiment
/home/core/kubernetes/NODIRSTUFF/EchoClient/echoclient 198.162.52.217:3333 1000 10 &

# terminate client by sending ctrl^c since it does not stop by itself. Wait before termination.
sleep $TOTAL_EXPERIMENT_TIME

# get PID of the client
PID=$!

# kill the client
kill $PID



#print variable to indicate finishing of the script.
echo $STRING
