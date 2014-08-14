#!/usr/bin/python

import arvados
import subprocess
import crunchutil.subst as subst
import shutil
import os
import sys

if len(arvados.current_task()['parameters']) > 0:
    p = arvados.current_task()['parameters']
else:
    p = arvados.current_job()['script_parameters']

t = arvados.current_task().tmpdir

os.unlink("/usr/local/share/bcbio-nextgen/galaxy")
os.mkdir("/usr/local/share/bcbio-nextgen/galaxy")
shutil.copy("/usr/local/share/bcbio-nextgen/config/bcbio_system.yaml", "/usr/local/share/bcbio-nextgen/galaxy")

with open("/usr/local/share/bcbio-nextgen/galaxy/tool_data_table_conf.xml", "w") as f:
    f.write('''<tables>
    <!-- Locations of indexes in the BWA mapper format -->
    <table name="bwa_indexes" comment_char="#">
        <columns>value, dbkey, name, path</columns>
        <file path="tool-data/bwa_index.loc" />
    </table>
    <!-- Locations of indexes in the Bowtie2 mapper format -->
    <table name="bowtie2_indexes" comment_char="#">
        <columns>value, dbkey, name, path</columns>
        <file path="tool-data/bowtie2_indices.loc" />
    </table>
    <!-- Locations of indexes in the Bowtie2 mapper format for TopHat2 to use -->
    <table name="tophat2_indexes" comment_char="#">
        <columns>value, dbkey, name, path</columns>
        <file path="tool-data/bowtie2_indices.loc" />
    </table>
    <!-- Location of SAMTools indexes and other files -->
    <table name="sam_fa_indexes" comment_char="#">
        <columns>index, value, path</columns>
        <file path="tool-data/sam_fa_indices.loc" />
    </table>
    <!-- Location of Picard dict file and other files -->
    <table name="picard_indexes" comment_char="#">
        <columns>value, dbkey, name, path</columns>
        <file path="tool-data/picard_index.loc" />
    </table>
    <!-- Location of Picard dict files valid for GATK -->
    <table name="gatk_picard_indexes" comment_char="#">
        <columns>value, dbkey, name, path</columns>
        <file path="tool-data/gatk_sorted_picard_index.loc" />
    </table>
</tables>
''')

os.mkdir("/usr/local/share/bcbio-nextgen/galaxy/tool-data")

with open("/usr/local/share/bcbio-nextgen/galaxy/tool-data/bowtie2_indices.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(dir $(bowtie2_indices))\n"))

with open("/usr/local/share/bcbio-nextgen/galaxy/tool-data/bwa_index.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(file $(bwa_index))\n"))

with open("/usr/local/share/bcbio-nextgen/galaxy/tool-data/gatk_sorted_picard_index.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(file $(gatk_sorted_picard_index))\n"))

with open("/usr/local/share/bcbio-nextgen/galaxy/tool-data/picard_index.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(file $(picard_index))\n"))

with open("/usr/local/share/bcbio-nextgen/galaxy/tool-data/sam_fa_indices.loc", "w") as f:
    f.write(subst.do_substitution(p, "index\tGRCh37\t$(file $(sam_fa_indices))\n"))

with open("/tmp/crunch-job/freebayes-variant.yaml", "w") as f:
    f.write('''
# Template for whole genome Illumina variant calling with FreeBayes
# This is a GATK-free pipeline without post-alignment BAM pre-processing
# (recalibration and realignment)
---
details:
  - analysis: variant2
    genome_build: GRCh37
    # to do multi-sample variant calling, assign samples the same metadata / batch
    # metadata:
    #   batch: your-arbitrary-batch-name
    algorithm:
      aligner: bwa
      mark_duplicates: true
      recalibrate: false
      realign: false
      variantcaller: freebayes
      platform: illumina
      quality_format: Standard
      # for targetted projects, set the region
      # variant_regions: /path/to/your.bed
''')

os.unlink("/usr/local/share/bcbio-nextgen/gemini_data")
os.symlink(arvados.get_job_param_mount("gemini_data"), "/usr/local/share/bcbio-nextgen/gemini_data")

os.chdir(arvados.current_task().tmpdir)

rcode = subprocess.call(["bcbio_nextgen.py", "--workflow", "template", "/tmp/crunch-job/freebayes-variant.yaml", "project1",
                         subst.do_substitution(p, "$(file $(R1))"),
                         subst.do_substitution(p, "$(file $(R2))")])

os.chdir("project1/work")

os.symlink("/usr/local/share/bcbio-nextgen/galaxy/tool-data", "tool-data")

rcode = subprocess.call(["bcbio_nextgen.py", "../config/project1.yaml", "-n", os.environ['CRUNCH_NODE_SLOTS']])

print("run-command: completed with exit code %i (%s)" % (rcode, "success" if rcode == 0 else "failed"))

if rcode == 0:
    os.chdir("../final")

    print("arvados-bcbio-nextgen: the follow output files will be saved to keep:")

    subprocess.call(["find", ".", "-type", "f", "-printf", "arvados-bcbio-nextgen: %12.12s %h/%f\\n"])

    print("arvados-bcbio-nextgen: start writing output to keep")

    done = False
    while not done:
        try:
            out = arvados.CollectionWriter()
            out.write_directory_tree(".", max_manifest_depth=0)
            outuuid = out.finish()
            api.job_tasks().update(uuid=arvados.current_task()['uuid'],
                                                 body={
                                                     'output':outuuid,
                                                     'success': (rcode == 0),
                                                     'progress':1.0
                                                 }).execute()
            done = True
        except Exception as e:
            print("arvados-bcbio-nextgen: caught exception: {}".format(e))
            time.sleep(5)

sys.exit(rcode)
