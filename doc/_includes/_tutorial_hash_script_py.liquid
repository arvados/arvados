#!/usr/bin/env python

# Import the hashlib module (part of the Python standard library) to compute md5.
import hashlib

# Import the Arvados sdk module
import arvados

# Get information about the task from the environment
this_task = arvados.current_task()

# Get the "input" field from "script_parameters" on the job creation object
this_job_input = arvados.getjobparam('input')

# Create the object access to the collection referred to in the input
collection = arvados.CollectionReader(this_job_input)

# Create an object to write a new collection as output
out = arvados.CollectionWriter()

# Set the name of output file within the collection
out.set_current_file_name("md5sum.txt")

# Get an iterator over the files listed in the collection
all_files = collection.all_files()

# Iterate over each file
for input_file in all_files:
    # Create the object that will actually compute the md5 hash
    digestor = hashlib.new('md5')

    while True:
        # read a 1 megabyte block from the file
        buf = input_file.read(2**20)

        # break when there is no more data left
        if len(buf) == 0:
            break

        # update the md5 hash object
        digestor.update(buf)

    # Get the final hash code
    hexdigest = digestor.hexdigest()

    # Get the file name from the StreamFileReader object
    file_name = input_file.name()

    # The "stream name" is the subdirectory inside the collection in which
    # the file is located; '.' is the root of the collection.
    if input_file.stream_name() != '.':
        file_name = os.join(input_file.stream_name(), file_name)

    # Write an output line with the md5 value and file name.
    out.write("%s %s\n" % (hexdigest, file_name))

# Commit the output to keep.  This returns a Keep id.
output_id = out.finish()

# Set the output for this task to the Keep id
this_task.set_output(output_id)

# Done!
