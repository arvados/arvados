#!/usr/bin/python

import arvados
import subprocess
import subst
import shutil
import os

if len(arvados.current_task()['parameters']) > 0:
    p = arvados.current_task()['parameters']
else:
    p = arvados.current_job()['script_parameters']

os.unlink("/usr/local/share/bcbio-nextgen/galaxy")
os.mkdir("/usr/local/share/bcbio-nextgen/galaxy")
shutil.copy("/usr/local/share/bcbio-nextgen/config/bcbio_system.yaml", "/usr/local/share/bcbio-nextgen/galaxy")

os.chdir(arvados.current_task().tmpdir)

with open("gatk-variant.yaml", "w") as f:
    f.write('''
# Template for whole genome Illumina variant calling with GATK pipeline
---
details:
  - analysis: variant2
    genome_build: GRCh37
    # to do multi-sample variant calling, assign samples the same metadata / batch
    # metadata:
    #   batch: your-arbitrary-batch-name
    algorithm:
      aligner: bwa
      mark_duplicates: picard
      recalibrate: gatk
      realign: gatk
      variantcaller: gatk-haplotype
      platform: illumina
      quality_format: Standard
      coverage_interval: genome
      # for targetted projects, set the region
      # variant_regions: /path/to/your.bed
''')

rcode = subprocess.call(["bcbio_nextgen.py", "--workflow", "template", "gatk-variant.yaml", "project1",
                         subst.do_substitution(p, "$(file $(R1))"),
                         subst.do_substitution(p, "$(file $(R2))")])

os.chdir("project1/work")

os.mkdir("tool-data")

with open("tool-data/bowtie2_indices.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(dir $(bowtie2_indices))"))

with open("tool-data/bwa_indices.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(file $(bwa_indices))"))

with open("tool-data/gatk_sorted_picard_index.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(file $(gatk_sorted_picard_index))"))

with open("tool-data/picard_index.loc", "w") as f:
    f.write(subst.do_substitution(p, "GRCh37\tGRCh37\tHuman (GRCh37)\t$(file $(picard_index))"))

with open("tool-data/sam_fa_indices.loc", "w") as f:
    f.write(subst.do_substitution(p, "index\tGRCh37\t$(file $(sam_fa_indices))"))

rcode = subprocess.call(["bcbio_nextgen.py", "../config/project1.yaml"])
