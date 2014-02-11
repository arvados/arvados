#!/usr/bin/env python

# Import the Arvados sdk module
import arvados

# Get information about the task from the environment
this_task = arvados.current_task()

this_task_input = arvados.current_job()['script_parameters']['input']

# Create the object access to the collection referred to in the input
collection = arvados.CollectionReader(this_task_input)

# Create an object to write a new collection as output
out = arvados.CollectionWriter()

# Set the name of output file within the collection
out.set_current_file_name("0-filter.txt")

# Get an iterator over the files listed in the collection
all_files = collection.all_files()

# Iterate over each file
for input_file in all_files:
    for ln in input_file.readlines():
        if ln[0] == '0':
            out.write(ln)

# Commit the output to keep.  This returns a Keep id.
output_id = out.finish()

# Set the output for this task to the Keep id
this_task.set_output(output_id)

# Done!
