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

<table>
<thead>
<tr class="header">
<th>Step</th>
<th>Who</th>
<th>What</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>0</td>
<td>engineering</td>
<td>Build new features, refine good code into great code</td>
</tr>
<tr class="even">
<td>1</td>
<td>ops</td>
<td><a href="https://ci.arvados.org/view/All/job/packer-build-compute-image/">Build a new tordo compute image</a> against the latest development packages.<br />
<a href="https://dev.arvados.org/projects/ops/wiki/Updating_clusters">Update the tordo configuration</a> and test it with a couple representative workflows (at least one bioinformatics workflow and one S3 download workflow).<br />
If everything works well, update version pins based on the versions installed in the new image. Update:</td>
</tr>
<tr class="odd">
<td>2</td>
<td>engineering</td>
<td>Prepare release branch on the <code>arvados</code> and <code>arvados-formula</code> repositories. For major releases, this means branching a new <code>X.Y-staging</code> from main. For minor releases, this means cherry-picking features onto the existing <code>X.Y-staging</code> branch. Ensure that Redmine issues for features or bugfixes that are appearing for the first time in this version are associated with the correct release (for major releases, use <code>art redmine issues find-and-associate</code>).</td>
</tr>
<tr class="even">
<td>3</td>
<td>engineering</td>
<td>Ensure that the release staging branch passes automated tests on Jenkins.</td>
</tr>
<tr class="odd">
<td>4</td>
<td>engineering</td>
<td>Review release branch to make sure all commits that need to be in the release are in the release. If new commits are added, resume checklist from step 3.</td>
</tr>
<tr class="even">
<td>5</td>
<td>product mgr</td>
<td>Write release notes and publish them <a href="https://www-dev.arvados.org/releases/">on the www-dev site</a>.</td>
</tr>
<tr class="odd">
<td>6</td>
<td>everyone</td>
<td>Review release notes</td>
</tr>
<tr class="even">
<td>7</td>
<td>product mgr</td>
<td>Create a Redmine release for the next patch release after the current one.</td>
</tr>
<tr class="odd">
<td>8</td>
<td>release eng</td>
<td>Build release candidate packages with version “X.Y.Z~rcN-1” using the Jenkins job <a href="https://ci.arvados.org/job/build-and-publish-rc-packages/">build-and-publish-rc-packages</a>. Add a comment on the release ticket identifying the Git commit hash used for the build, and link to your Jenkins run.</td>
</tr>
<tr class="even">
<td>9</td>
<td>release eng</td>
<td>Publish release candidate <code>arvados/jobs</code> Docker image using <a href="https://ci.arvados.org/job/docker-jobs-image-release/">docker-jobs-image-release</a></td>
</tr>
<tr class="odd">
<td>10</td>
<td>ops</td>
<td>Test installer formula / provision scripts with RC packages. Run the <a href="https://ci.arvados.org/job/test-provision/">test-provision Jenkins job</a> where <code>git_hash</code> is your <code>X.Y-staging</code> commit and <code>RELEASE</code> is <code>testing</code>.</td>
</tr>
<tr class="even">
<td>11</td>
<td>ops</td>
<td>Update pirca to use the RC packages: <a href="https://ci.arvados.org/job/packer-build-compute-image/">build a new compute image</a>, <a href="https://dev.arvados.org/projects/ops/wiki/Updating_clusters">update the Arvados version in Salt</a> and deploy.<br />
After Salt updates the cluster, check that your new version deployed successfully by running <code>arvados-server version</code> and then <code>arvados-server check</code> to verify other running services have the same version.</td>
</tr>
<tr class="odd">
<td>12</td>
<td>bfx</td>
<td>Run <a href="https://ci.arvados.org/job/run-tests-cwl-suite/">CWL integration tests</a> and <a href="https://workbench.pirca.arvadosapi.com/workflows/pirca-7fd4e-ut5n6r2ydl6o6kj">fastq-to-gvcf pipeline</a> on pirca ([[more about running fastq-to-gvcf]]).<br />
After the workflow succeeds, check the versions reported at the top of the workflow logs to verify it ran your RC for crunch-run, arv-mount, and a-c-r.</td>
</tr>
<tr class="even">
<td>13</td>
<td>engineering</td>
<td>Perform final manual testing based on risk assessment, the release notes and <a href="https://dev.arvados.org/projects/arvados/wiki/Manual_testing_plan">manual testing plan</a>. This should involve at least a “smell check” to confirm that key features, improvements or bug fixes intended to appear in the release are present and behave as intended.</td>
</tr>
<tr class="odd">
<td>14</td>
<td>product mgr</td>
<td>Approve RC for release</td>
</tr>
<tr class="even">
<td>15</td>
<td>release eng</td>
<td>Publish Ruby gems using <a href="https://ci.arvados.org/job/build-publish-packages-python-ruby/">build-publish-packages-python-ruby</a> with <strong>only</strong> the <code>BUILD_RUBY</code> box checked.</td>
</tr>
<tr class="odd">
<td>16</td>
<td>release eng</td>
<td>On the <code>X.Y-staging</code> branch, update these files to refer to the release version: </td>
</tr>
<tr class="even">
<td>17</td>
<td>release eng</td>
<td>Build final release packages with version “X.Y.Z-1” using the Jenkins job <a href="https://ci.arvados.org/job/build-and-publish-rc-packages/">build-and-publish-rc-packages</a>. Add a comment on the release ticket identifying the Git commit hash used for the build, and link to your Jenkins run.</td>
</tr>
<tr class="odd">
<td>18</td>
<td>release eng</td>
<td>Publish stable release <code>arvados/jobs</code> Docker image using <a href="https://ci.arvados.org/job/docker-jobs-image-release/">docker-jobs-image-release</a></td>
</tr>
<tr class="even">
<td>19</td>
<td>release eng</td>
<td>Push packages to stable repos using <a href="https://ci.arvados.org/job/publish-packages-to-stable-repo/">publish-packages-to-stable-repo</a> (<a href="https://dev.arvados.org/projects/ops/wiki/Promoting_Packages_to_Stable)">more info</a></td>
</tr>
<tr class="odd">
<td>20</td>
<td>release eng</td>
<td>Publish Python packages using <a href="https://ci.arvados.org/job/build-publish-packages-python-ruby/">build-publish-packages-python-ruby</a> with <strong>only</strong> the <code>BUILD_PYTHON</code> box checked.</td>
</tr>
<tr class="even">
<td>21</td>
<td>release eng</td>
<td>Publish Java package using <a href="https://ci.arvados.org/job/build-java-sdk/">build-java-sdk</a> and following [[Releasing Java SDK packages]]</td>
</tr>
<tr class="odd">
<td>22</td>
<td>release eng</td>
<td>Publish R package using <a href="https://ci.arvados.org/job/build-package-r/">build-package-r</a></td>
</tr>
<tr class="even">
<td>23</td>
<td>release eng</td>
<td>Tag the commits in each repo used to build the release in Git. Create an annotated tag (<code>git tag --annotate</code>) with a message like “Release notes at https://arvados.org/release-notes/X.Y.Z/” That makes the <a href="https://github.com/arvados/arvados/releases">GitHub releases page</a> look good. See <a href="https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes">GitHub documentation for more details about how to automate releases</a>.<br />
Create or fast forward the <code>X.Y-release</code> branch to match <code>X.Y-staging</code>.<br />
Cherry-pick the upgrade notes commit (from step 2) onto <code>main</code>.</td>
</tr>
<tr class="odd">
<td>24</td>
<td>release eng</td>
<td>Ensure new release is published on https://doc.arvados.org/.<br />
Ensure that release notes &amp; any other materials are pointing to correct version of the docs.<br />
(If anything goes wrong, see https://dev.arvados.org/projects/arvados-private/wiki/Docarvadosorg_deployment)</td>
</tr>
<tr class="even">
<td>25</td>
<td>ops</td>
<td>Update pirca and jutro to the new stable release: <a href="https://ci.arvados.org/job/packer-build-compute-image/">build new compute images</a>, <a href="https://dev.arvados.org/projects/ops/wiki/Updating_clusters">update the Arvados version in Salt</a> and deploy.</td>
</tr>
<tr class="odd">
<td>26</td>
<td>product mgr</td>
<td>Merge release notes (step 6) from “develop” branch to “main” branch of the <code>arvados-www</code> Git repository and check that the https://arvados.org front page is updated</td>
</tr>
<tr class="even">
<td>27</td>
<td>product mgr</td>
<td>Send out the release notes via MailChimp, tweet from the Arvados account, announce on the Discourse forum, Matrix, etc.</td>
</tr>
<tr class="odd">
<td>28</td>
<td>release eng</td>
<td>In Jenkins:</td>
</tr>
<tr class="even">
<td>29</td>
<td>release eng</td>
<td>Add the release to <a href="https://doi.org/10.5281/zenodo.6382942">doi:10.5281/zenodo.6382942</a><br />
[[Updating Zenodo Version of Arvados after Release]]<br />
https://zenodo.org/record/6382943</td>
</tr>
</tbody>
</table>
