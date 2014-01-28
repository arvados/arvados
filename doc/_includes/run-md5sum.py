#!/usr/bin/env python

import arvados

arvados.job_setup.one_task_per_input_file(if_sequence=0, and_end_task=True)
this_task = arvados.current_task()

# Get the input collection for this task
this_task_input = this_task['parameters']['input']

# Create a CollectionReader to access the collection
input_collection = arvados.CollectionReader(this_task_input)

# Get the name of the first file in the collection
input_file = list(input_collection.all_files())[0].name()

# Extract the file to a temporary directory
# Returns the directory that the file was written to
input_dir = arvados.util.collection_extract(this_task_input,
        'tmp',
        files=[input_file],
        decompress=False)

# Run the external 'md5sum' program on the input file, with the current working
# directory set to the location the input file was extracted to.
stdoutdata, stderrdata = arvados.util.run_command(
        ['md5sum', input_file],
        cwd=input_dir)

# Save the standard output (stdoutdata) "md5sum.txt" in the output collection
out = arvados.CollectionWriter()
out.set_current_file_name("md5sum.txt")
out.write(stdoutdata)

this_task.set_output(out.finish())
