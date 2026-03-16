[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# More about running fastq-to-gVCF

When we do releases, we run a test pipeline that is intended to be representative of a bioinformatics workload.

1. Deploy the version of `arvados-cwl-runner` that you want to test and make sure that the corresponding `arvados/jobs` image [has been built and uploaded to docker hub](https://ci.arvados.org/view/Release%20Pipeline/job/docker-jobs-image-release/) or built using the `arvados/build/build-dev-docker-jobs-image.sh` script and uploaded using `arv-keepdocker`.
2. Clone https://git.arvados.org/arvados-tutorial.git/
3. Create an Arvados project for the test run
4. `cd arvados/tutorial/WGS-processing`
5. Run the following command: `arvados-cwl-runner --no-wait --disable-reuse --project-uuid <my project> cwl/wgs-processing-wf.cwl yml/wgs-processing-wf-chr19.yml`
6. Monitor this for success. It usually takes about an hour to run.

If you are running this on `pirca` then all the data should already be present. If you are running it from somewhere else, you may need to do some additional data copying from `pirca` to the other cluster. The input document `yml/wgs-processing-wf-chr19.yml` has the portable data hashes of the collections.
