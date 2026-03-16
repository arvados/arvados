[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Release Checklist

Pre-process:

1.  Create an issue for the release.
2.  Add each of the following steps (starting at step 1) as tasks with the step number in the subject
3.  Assign each task
4.  The current task goes into the “In Progress” column
5.  When the current task is finished, move it to resolved, and move the next task into “In Progress”
6.  Notify the assignee of the next task that it is ready to begin

Meta-process:

1.  Periodically review this documented process reflects our actual process & update it
2.  When steps are added/changed/rearranged/removed, be sure to update [`cmd/art/TASKS` in the `arvados-dev` repository](https://dev.arvados.org/projects/arvados/repository/arvados-dev/revisions/main/show/cmd/art).

| Step | Who         | What                                                 |
|------|-------------|------------------------------------------------------|
|   0  | engineering | Build new features, refine good code into great code |
|   1  | ops         | [Build a new tordo compute image](https://ci.arvados.org/view/All/job/packer-build-compute-image/) against the latest development packages.<br>[Update the tordo configuration](https://dev.arvados.org/projects/ops/wiki/Updating_clusters) and test it with a couple representative workflows (at least one bioinformatics workflow and one S3 download workflow).<br>If everything works well, update version pins based on the versions installed in the new image. Update:<br>* `tools/ansible/roles/arvados_docker/files/arvados-docker.pref`<br>* `tools/ansible/roles/compute_amd_rocm/defaults/main.yml` (update `arvados_compute_amd_rocm_version`)<br>* `tools/ansible/roles/compute_nvidia/files/arvados-nvidia.pref` |
|   2  | engineering | Prepare release branch on the `arvados` and `arvados-formula` repositories. For major releases, this means branching a new `X.Y-staging` from main. For minor releases, this means cherry-picking features onto the existing `X.Y-staging` branch. Ensure that Redmine issues for features or bugfixes that are appearing for the first time in this version are associated with the correct release (for major releases, use `art redmine issues find-and-associate`). |
|   3  | engineering | Ensure that the release staging branch passes automated tests on Jenkins.<br>* [developer-run-tests](https://ci.arvados.org/job/developer-run-tests/)<br>* [developer-run-tests-doc-sdk-java-R](https://ci.arvados.org/job/developer-run-tests-doc-sdk-java-R/)<br>* [arvados-cwl-conformance-tests](https://ci.arvados.org/job/arvados-cwl-conformance-tests/) |
|   4  | engineering | Review release branch to make sure all commits that need to be in the release are in the release. If new commits are added, resume checklist from step 3. |
|   5  | product mgr | Write release notes and publish them [on the www-dev site](https://www-dev.arvados.org/releases/). |
|   6  | everyone    | Review release notes |
|   7  | product mgr | Create a Redmine release for the next patch release after the current one. |
|   8  | release eng | Build release candidate packages with version `X.Y.Z~rcN-1` using the Jenkins job [build-and-publish-rc-packages](https://ci.arvados.org/job/build-and-publish-rc-packages/). Add a comment on the release ticket identifying the Git commit hash used for the build, and link to your Jenkins run. |
|   9  | release eng | Publish release candidate `arvados/jobs` Docker image using [docker-jobs-image-release](https://ci.arvados.org/job/docker-jobs-image-release/) |
|   10 | ops         | Test installer formula / provision scripts with RC packages. Run the [test-provision Jenkins job](https://ci.arvados.org/job/test-provision/) where `git_hash` is your `X.Y-staging` commit and `RELEASE` is `testing`. |
|   11 | ops         | Update pirca to use the RC packages: [build a new compute image](https://ci.arvados.org/job/packer-build-compute-image/), [update the Arvados version in Salt](https://dev.arvados.org/projects/ops/wiki/Updating_clusters) and deploy.<br>After Salt updates the cluster, check that your new version deployed successfully by running `arvados-server version` and then `arvados-server check` to verify other running services have the same version. |
|   12 | bfx         | Run [CWL integration tests](https://ci.arvados.org/job/run-tests-cwl-suite/) and [fastq-to-gvcf pipeline](https://workbench.pirca.arvadosapi.com/workflows/pirca-7fd4e-ut5n6r2ydl6o6kj) on pirca ([more about running fastq-to-gvcf](/projects/arvados/wiki/More_about_running_fastq-to-gvcf)).<br>After the workflow succeeds, check the versions reported at the top of the workflow logs to verify it ran your RC for crunch-run, arv-mount, and a-c-r. |
|   13 | engineering | Perform final manual testing based on risk assessment, the release notes and [manual testing plan](https://dev.arvados.org/projects/arvados/wiki/Manual_testing_plan). This should involve at least a "smell check" to confirm that key features, improvements or bug fixes intended to appear in the release are present and behave as intended. |
|   14 | product mgr | Approve RC for release |
|   15 | release eng | Publish Ruby gems using [build-publish-packages-python-ruby](https://ci.arvados.org/job/build-publish-packages-python-ruby/) with **only** the `BUILD_RUBY` box checked. |
|   16 | release eng | On the `X.Y-staging` branch, update these files to refer to the release version:<br>* `doc/admin/upgrading.html.textile.liquid` the "Upgrading Arvados and Release notes" doc page with the version and date of the release.<br>* `contrib/arvados-bootstrap/pyproject.toml`, update `project.version` and `project.dependencies`<br>* `contrib/R-sdk/DESCRIPTION`, update `Version:`<br>* `services/api/Gemfile` to depend on the newly published Arvados gem and run `bundle install` to update `Gemfile.lock`<br>* `tools/ansible/roles/arvados_apt/defaults/main.yml` update `arvados_pin_version` |
|   17 | release eng | Build final release packages with version `X.Y.Z-1` using the Jenkins job [build-and-publish-rc-packages](https://ci.arvados.org/job/build-and-publish-rc-packages/). Add a comment on the release ticket identifying the Git commit hash used for the build, and link to your Jenkins run. |
|   18 | release eng | Publish stable release `arvados/jobs` Docker image using [docker-jobs-image-release](https://ci.arvados.org/job/docker-jobs-image-release/) |
|   19 | release eng | Push packages to stable repos using [publish-packages-to-stable-repo](https://ci.arvados.org/job/publish-packages-to-stable-repo/) ([more info](https://dev.arvados.org/projects/ops/wiki/Promoting_Packages_to_Stable)) |
|   20 | release eng | Publish Python packages using [build-publish-packages-python-ruby](https://ci.arvados.org/job/build-publish-packages-python-ruby/) with **only** the `BUILD_PYTHON` box checked. |
|   21 | release eng | Publish Java package using [build-java-sdk](https://ci.arvados.org/job/build-java-sdk/) and following [Releasing Java SDK packages](/projects/arvados/wiki/Releasing_Java_SDK_packages) |
|   22 | release eng | Publish R package using [build-package-r](https://ci.arvados.org/job/build-package-r/) |
|   23 | release eng | Tag the commits in each repo used to build the release in Git. Create an annotated tag (`git tag --annotate`) with a message like "Release notes at https://arvados.org/release-notes/X.Y.Z/" That makes the [GitHub releases page](https://github.com/arvados/arvados/releases) look good. See [GitHub documentation for more details about how to automate releases](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes).<br>Create or fast forward the `X.Y-release` branch to match `X.Y-staging`.<br>Cherry-pick the upgrade notes commit (from step 2) onto `main`. |
|   24 | release eng | Ensure new release is published on https://doc.arvados.org/.<br>Ensure that release notes & any other materials are pointing to correct version of the docs.<br>(If anything goes wrong, see https://dev.arvados.org/projects/arvados-private/wiki/Docarvadosorg_deployment) |
|   25 | ops         | Update pirca and jutro to the new stable release: [build new compute images](https://ci.arvados.org/job/packer-build-compute-image/), [update the Arvados version in Salt](https://dev.arvados.org/projects/ops/wiki/Updating_clusters) and deploy. |
|   26 | product mgr | Merge release notes (step 6) from "develop" branch to "main" branch of the `arvados-www` Git repository and check that the https://arvados.org front page is updated |
|   27 | product mgr | Send out the release notes via MailChimp, tweet from the Arvados account, announce on the Discourse forum, Matrix, etc. |
|   28 | release eng | In Jenkins:<br>* For each test from step 3, go to "Job Config History" and record on the release ticket the timestamp of the configuration used to test the release<br>* Go to [Manage Jenkins > Clouds > gce2 > Configure](https://ci.arvados.org/manage/cloud/gce-gce2/configure) and record the VM image tagged "tests" used for jenkins workers to run the tests for the release (should be something like jenkins-image-arvados-tests-YYYYMMDDHHMMSS) on the release ticket<br>* Go to [packer-build-jenkins-image-arvados-tests history](https://ci.arvados.org/job/packer-build-jenkins-image-arvados-tests/) and record on the release ticket the Jenkins job used to build the above VM image. |
|   29 | release eng | Add the release to [doi:10.5281/zenodo.6382942](https://doi.org/10.5281/zenodo.6382942)<br>[Updating Zenodo Version of Arvados after Release](Zenodo.md)<br>[https://zenodo.org/record/6382943](https://zenodo.org/record/6382943) |
