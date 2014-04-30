#! /bin/sh

# Stress test for Keep.
#
# This test generates a specified number of 64MB data files of random
# data, then starts a Keep server on localhost and launches a number
# of concurrent ab requests against it.
#
# Configuration settings:
#   NUMREQUESTS
#       Total number of iterations to perform for each
#       request. (-n option to ab)
#   MAXCONCURRENT
#       Maximum number of concurrent requests per process.
#       (-c option to ab)
#   NUMFILES
#       Number of data files to generate for the test.

NUMREQUESTS=100
MAXCONCURRENT=5
NUMFILES=3

DATADIR=$HOME/keeploadtest.$$
mkdir $DATADIR || exit 1

# Generate random data files.
for i in $(seq 1 $NUMFILES)
do
    echo "Generating file #$i..."
    head --bytes=64M /dev/urandom > $DATADIR/data
    md5=$(md5sum $DATADIR/data | awk '{print $1}')
    mv $DATADIR/data $DATADIR/$md5
done

# start keep
keep1=$HOME/keepdir.$$.1
keep2=$HOME/keepdir.$$.2
mkdir $keep1
mkdir $keep2
echo "Starting keep..."
bin/keep -volumes=$keep1,$keep2 > /tmp/keep.log 2>&1 &

# run benchmarks
for loc in $(ls $DATADIR)
do
    # PUT request
    ab -u $DATADIR/$loc -n $NUMREQUESTS -c $MAXCONCURRENT http://localhost:25107/$loc &
    # GET request
    ab -n $NUMREQUESTS -c $MAXCONCURRENT http://localhost:25107/$loc &
done

