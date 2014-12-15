#!/usr/bin/env python

import hashlib      # Import the hashlib module to compute MD5.
import os           # Import the os module for basic path manipulation
import arvados      # Import the Arvados sdk module

# Automatically parallelize this job by running one task per file.
# This means that if the input consists of many files, each file will
# be processed in parallel on different nodes enabling the job to
# be completed quicker.
arvados.job_setup.one_task_per_input_file(if_sequence=0, and_end_task=True,
                                          input_as_path=True)

# Get object representing the current task
this_task = arvados.current_task()

# Create the message digest object that will compute the MD5 hash
digestor = hashlib.new('md5')

# Get the input file for the task
input_id, input_path = this_task['parameters']['input'].split('/', 1)

# Open the input collection
input_collection = arvados.CollectionReader(input_id)

# Open the input file for reading
with input_collection.open(input_path) as input_file:
    for buf in input_file.readall():  # Iterate the file's data blocks
        digestor.update(buf)          # Update the MD5 hash object

# Write a new collection as output
out = arvados.CollectionWriter()

# Write an output file with one line: the MD5 value and input path
with out.open('md5sum.txt') as out_file:
    out_file.write("{} {}/{}\n".format(digestor.hexdigest(), input_id,
                                       os.path.normpath(input_path)))

# Commit the output to Keep.
output_locator = out.finish()

# Use the resulting locator as the output for this task.
this_task.set_output(output_locator)

# Done!
