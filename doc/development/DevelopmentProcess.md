[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Summary of Development Process

{{toc}}

# Filing bugs and task scheduling

The Arvados project uses Redmine for organizing our work. You are probably looking at the Redmine site right now.

## Where to create the ticket

Add issue using the “Issues” tab.

Alternately, from the “Backlogs” interface, go to “Product backlog” and then “New story”.

## Issue trackers

When filing an issue, use these guidelines to choose how it should be tracked:

|         |                                                                                                                                                                                                                                                                                         |
|---------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Bug     | A flaw where the software does not behave as intended, or misleading/outdated documentation.                                                                                                                                                                                            |
| Feature | A well defined task taken to improve the software or documentation. This should be fully specified and actionable, otherwise use “Idea”.                                                                                                                                                |
| Support | An non-development action that needs to be performed to assist a user or customer, or an ops task to maintain internal or external systems.                                                                                                                                             |
| Task    | A task is a smaller unit of work attached to a feature or bug fix, most commonly used to communicate the state of code review on the task board. **Note** this shows up in the “Tracker” drop-down for top level tickets but shouldn’t be used unless you are filling in parent ticket. |
| Idea    | An idea, concept, proposal or project to improve the software or documentation, which needs additional work to fully specify and/or broken down into concrete steps.                                                                                                                    |

## Issue category

We use “category” to describe the part of the product most relevant to the bug or feature. It is used to ensure that tickets are assigned to developers who are knowledgeable about that part of the product.

## Writing good ticket descriptions

When submitting a ticket, you should aim to outline:

- Background / context
- Current behavior
- Desired fixes / improvements

If you have ideas about how the issue could be addressed, feel free to add:

- Proposed implementation
- Exclusions / clarifications
- Open questions

These can be added or refined as part of later grooming.

## Ticket triage

At least once a week, the product manager will go through new tickets in “Product backlog”. (Maybe we should do this at a regular meeting?)

If not done already, the ticket should be linked (using “Related to”) to an “epic” or “request (tracking)” ticket (these are “idea” tickets). If a ticket doesn’t relate to any existing epics, it may represent a new project that needs to be added to “Epics”.

If a ticket is deemed sufficiently urgent/high priority, it will be scheduled for an upcoming sprint.

Otherwise, following initial triage, the ticket should be added to the “Future” sprint. This is the holding area for tickets that are not scheduled. Because there are a very large number of tickets in this state, it is important to properly link tickets to epics or customer requests so they can be found again.

## Ticket scheduling

If a ticket did not jump the line and be scheduled on a sprint, it will be pulled in during either the off week engineering meeting or during sprint preparation (the meeting on Tuesday the day before sprint turnover). During these meeting we examine the current epics, customer priorities, and other internal priorities, look at tickets linked to those, and schedule them on an upcoming sprint.

# Revision control

## Branches

- All development should be done in a branch. The only exception to this should be trivial bug fixes. What is trivial enough to not need review is the judgement of the developer, but when in doubt, ask for a review.
- Each story should be done in its own branch.
- Branch names are “\####-story-summary” where \#### is the redmine issue number followed by 3 or 4 words that summarize the story.
- Make your local branches track the main repository (`git push -u`)
- Commit regularly, and push your branch to `git.arvados.org` at the end of each day
  - Be paranoid, commits are cheap, pushing your commits to the remote repository is cheap, losing work is expensive
  - The preferred format of a commit message on a branch is like this (where 12345 should be replaced by the redmine issue number):
        12345: One line summary of changes in this commit

        More detailed description of changes if relevant.

        Arvados-DCO-1.1-Signed-off-by: Your Name <your.email@curii.com>
- Don’t push uninvited changes to other developer’s branches.
  - To contribute to another developer’s branch, check with them first, or create your own branch (“\####-story-summary-ABC” where ABC are your initials) and ask the other developer to merge your branch.

### Merging

Branches should not be merged to main until they are ready (see \[\[#Ready to merge\]\] below).

1.  `git remote -v`
    - Make sure your `origin` is git.arvados.org, not github. **Don’t push directly to the github main** branch — let git.arvados.org decide whether it’s OK to push to github.
2.  `git checkout main`
3.  `git pull --ff-only`  
    \#\* This ensures your main is up to date. Otherwise “git push” below might fail, and you’ll be backtracking.
4.  `git merge --no-ff branchname`  
    \#\* **The `--no-ff` part is important!** It ensures there is actually a commit representing this merge. This is your opportunity to record the name of your branch being merged, and the relevant story number. Without it, the git history looks like we all just mysteriously started developing at the tip of your (now unnamed) feature branch.  
    \#\* In your merge commit message, **include the relevant story/issue number** (either “`refs #1234`” or “`closes #1234`”).  
    \#\* In your merge commit message, **include Arvados-DCO-1.1-Signed-off-by line** (i.e. Arvados-DCO-1.1-Signed-off-by: Jane Doe \<jane@example.com\>)  
    \#\* The preferred format of a merge commit message is like this:
        Merge branch '12345-story-summary'

        refs #12345

        Arvados-DCO-1.1-Signed-off-by: Your Name <your.email@curii.com>
5.  `git push`
6.  Look for Jenkins’ build results at https://ci.arvados.org .

### Rejected pushes

We have a git hook in place that will reject pushes that do not follow these guidelines. The goal of these policies is to ensure a clean linear history of changes to main with consistent cross referencing with issue numbers. These policies apply to the commits listed on “git rev-list —first-parent” when pushing to main, and not to commits on any other branches.

If you try to push a (set of) commit(s) that does not pass mustard, you will get a \[POLICY\] reject message on stdout, which will also list the offending commit. You can use

git commit —amend

to update the commit message on your last commit, if that is the offending one, or else you can use

git rebase —interactive

to rebase and fix up a commit message on an earlier commit.

#### All merge commits to main must be from a feature branch into main

Merges that go the other way (from main to a feature branch) that get pushed to main as a result of a fast-forward push will be rejected. In other words: when merging to main, make sure to use —no-ff.

#### Merges between local and remote main branches will be rejected

Merges between local and remote main branches (generally merges created by “git pull”) will be rejected, in order to maintain a linear main history. If this happens, you’ll need to reset main to the remote head and then remerge or rebase.

#### Proper merge message format

All merge commits to main must include the text “Merge branch ‘featurebranch’” or they will be rejected.

#### All commits to main include an issue number or explicitly say “no issue \#”

All commits to main (both merges and single parent commits) must  
include the text “refs \#”, “closes \#”, “fixes \#”, or “no issue \#” or they will be  
rejected.

#### Avoid broken commit messages

Your commit message matches

/Please enter a commit message to explain why this merge is necessary/

## Commit logs

See https://dev.arvados.org/projects/arvados/wiki/Coding_Standards

## Code review process

Code review has high priority! Branches shouldn’t sit around for days waiting for review/merge.

When your branch is ready for review:

1.  Create/update a review task on the story so it looks like this:  
    \#\* subject = “review {branch name}”  
    \#\* state = in progress  
    \#\* assignee is not null
2.  Comment on the issue page (not the review page), including  
    \#\* branch name  
    \#\* commit hash  
    \#\* link to Jenkins test run  
    \#\* if appropriate, a brief description of what’s in the branch (may be omitted if it’s already in the commit messages, or if it would just be a copy of the issue subject/description)
3.  Ping your reviewer (during daily standup, via e-mail and/or via chat).

Doing a review:

1.  Reviewers are usually assigned at sprint kickoff, but if you don’t have a reviewer, ask for a volunteer in chat and/or at daily stand-up.
2.  When you start the review, assign the review task to yourself and move the review task to “in progress” to make sure other people don’t duplicate your effort.
3.  The recommended process for reviewing diffs for a branch is `git diff main...branchname`. The reviewer must make sure that their repository is up to date (or use `git diff origin/main...origin/branchname`). Note the 3 dots (not two)
4.  The reviewer goes through the \[\[#Ready-to-merge\]\] checklist
5.  After doing a review, write up comments (“fix these problems” or “ready to merge”) to the story page, make a note of the git commit revision that was reviewed, assign the review task back to the original developer, and notify the original developer by chat (or by some other means such as at daily standup).  
    \#\* In comments, it is helpful to indicate how strongly you feel how/important the comment is as “low”, “medium”, or “high”  
    \#\* low: nitpick not necessarily worth changing here if you don’t feel like it, but I’m mentioning it to help improve habits  
    \#\* medium: suggestion/idea that you should at least acknowledge/respond to, even if we don’t end up resolving it here  
    \#\* high: we should make sure we both agree on how this is resolved before merging
6.  The original developer should address any outstanding problems/comments in the code, then write a brief response indicating which points were dealt with or intentionally rejected/not addressed.
7.  If the response involves more commits, do that, then goto “branch is ready for review”. This process iterates until the branch is deemed ready to merge (indicating by posting a comment with “LGTM” for “Looks Good To Merge”)
8.  When the comments are all low priority, someone might write something like “LGTM if you fix this one typo”, this indicates that once the minor comments are handled (fixed or responded to) that the branch should be merged without another review cycle.
9.  Once the branch is merged, move the “review” task to “resolved”.

To list unmerged branches:

- Yours: `git branch --no-merged main`
- Everyone: `git branch -a --no-merged main`

## Ready to merge

See also the [Ready-to-merge-checklist](https://dev.arvados.org/projects/arvados/wiki/Coding_Standards#Ready-to-merge-checklist)

When merging, both the developer and the reviewer should be convinced that:

- Current/recent main is merged. (Otherwise, you can’t predict what merge will do.)
- The branch is pushed to git.arvados.org
- The code is suitably robust.
- The code is suitably readable.
- The code is suitably scalable. For example, client code is not allowed to print or sort unbounded lists. If the code handles a list of items, consider what happens when the list is 10x as large as you expect. What about 100x? A million times?
- The code accomplishes what the story specified. If not, explain why (e.g., the branch is only part of the story, a better solution was found, etc.) in the issue comments
- New API names (methods, attributes, error codes) and behaviors are well chosen. It sucks to change them later, and have to choose between compatibility and greatness.
- Tests that used to pass still pass. (Be extremely careful when altering old tests to make them pass. Do not change existing tests to test new code. Add assertions and write new tests. If you change or remove an existing test, you are breaking behavior that someone already decided was worth testing!)
- Recent clients/SDKs work against the new API server. (Things rarely turn out well when we rely on all clients being updated at once in lockstep with the API server. Our test suite doesn’t check this for us yet, so for now we have to pay attention.)
- New/fixed behavior is tested. (Although sometimes we decide not to block on inadequate testing infrastructure… that sucks!)
- New/changed behavior is documented. Search the doc site for relevant keywords to help you find the right sections.
- Whitespace errors are not committed. (Tab characters, spaces at EOL, etc.)
- Git commit messages are descriptive. If they aren’t, this is your last chance to rebase/reword.
- Code meets other \[\[arvados:Coding Standards\]\]
- For GUI work: user interface elements meet accessibility guidelines on the coding standards page

## Handling pull requests from github

*This is only for contributions by **external contributors**, i.e., people who don’t have permission to write directly to arvados.org repositories.*

First make sure your main is up to date.

git checkout main; git pull —ff-only

**Option 1:** On the pull request page on github, click the “You can also merge branches on the command line” link to get instructions.

- Don’t forget to run tests.

**Option 2:** (a bit shorter)

Say we have “chapmanb wants to merge 1 commit into arvados:main from chapmanb:branchname”

- `git fetch https://github.com/chapmanb/arvados.git branchname:chapmanb-branchname`
- `git merge --no-ff chapmanb-branchname`
- Use the commit message: `Merge branch 'branchname' from github.com/chapmanb. No issue #`  
  (or `refs #1234` if there is an issue#)
- Confirm diff: `git diff origin/main main`
- Run tests
- `git push`

# Non-fast-forward push

Please don’t get into a situation where this is needed.

1.  On dev box: `git push -f git`github.com:arvados/arvados proper_head_commit:main proper_head_commit:staging@
2.  On dev box: `git push -f git`git.arvados.org:arvados.git proper_head_commit:main@
3.  As gitsync@dev.arvados.org: `cd /scm/arvados; git fetch origin; git checkout main; git reset --hard origin/main`

(At least that’s what TC did on 2016-03-10. We’ll see how it goes.)

# Working with external upstream projects

Development process summary (1-6 should follow the guidelines above)

1.  Each feature is developed in a git branch named `<issue_number>-<summary>`, for example `12521-web-app-config`
2.  Each feature has a “Review” task. You can see the features and review tasks on the task board.
3.  When the feature branch is ready for review, update the title of the Review task to say “Review <branchname>” and move it from the **New** column the to **In Progress** column
4.  The reviewer responds on the issue page with questions or comments
5.  When the branch is ready to merge, the reviewer will add a comment “Looks Good To Me” (LGTM) on the issue page
6.  Merge the feature into into the Arvados main branch
7.  Push the feature branch to github and make a pull request (PR) of the branch against the external project upstream
8.  Handle code review comments/change requests from the external project team
9.  Once the external project merges the PR, merge external project upstream main back into the feature branch
10. Determine if external project upstream brings any unrelated changes that breaks things for us
11. If necessary, make fixes, make a new PR, repeat until stable
12. Merge the feature branch (now up-to-date with external project upstream) into Arvados main

This process is intended to let us work independently of how quickly the external project team merges our PRs, while still maximizing the chance that they will be able to accept our PRs by limiting the scope to one feature at a time.

This assumes using git merge commits and avoiding rebases, so we can easily perform merges back and forth between the three branches (Arvados main, feature, external project main).

# Scrum

## References

These books give us a reference point and vocabulary.

- *Essential Scrum: A Practical Guide to the Most Popular Agile Process* by Kenneth Rubin
- *User Stories Applied: For Agile Software Development* by Mike Cohen

## Roles

### Product Owner

- Decide what goes on the backlog
- Decide backlog priorities
- Work with stakeholders to understand their requirements and priorities
- Encode stakeholder requirements/expectations as user stories
- Lead sprint planning meetings
- Lead release planning meetings
- Lead product planning meetings
- Lead Sprint Kick-off Meetings
- Lead Sprint Review Meetings
- Decide on overall release schedule

### Scrum Master

- Lead Daily Scrum Meeting
- Help to eliminate road blocks
- Lead Sprint Retrospective Meetings
- Organize Sprint Schedule
- Help team organize and stay on track with Scrum process
- Teach new engineers how Scrum works

### Top stakeholders

- Conduct market research
- Synthesize market research into user stories
- Work with Product Owner to prioritize stories
- Define overall business goals for product
- Work with Product Owner to define overall release cycle
- Organize User Input and dialog with users for engineering team
- Contribute to backlog grooming
- Bring voice of customer into planning process
- Define user personas
- Coordinate user communication
- Develop technical marketing and sales materials
- Assist sales team in presenting product value proposition
- Train sales in technical aspects of the product

## Definition of Done

An issue is resolved when:

- Code is written
- Existing code is refactored if appropriate
- Documentation is written/updated
- Acceptance tests are satisfied
- Code is merged in main
- All Jenkins jobs pass (test, build packages, deploy to dev clusters)
- Feature works on applicable dev clusters

## Standard Schedule

Sprints are two weeks long. They start and end on Wednesdays.

### Key meetings

Every day:

Daily Scrum (15 Minutes)  
Who: Development team, product owner. Silent observers welcome.

- What did you do yesterday?
- What will you do today?
- What obstacles are in your way?

#### Sprint review & kickoff (every 2 weeks on Wednesday):

Sprint Review (30 minutes)  
Who: Development team, product owner, stakeholders.

- Demo of each feature built and relationship to stories
- Product owner explains which backlog items are done
- Development team demonstrates the work done, and answers questions about the sprint increment
- Product owner discusses the backlog as it stands. Revise expected completion dates based on recent progress (if needed)
- Review current product status in context of business goals

Sprint Retrospective (30 minutes)  
Who: Development team, product owner.

- Review what processes worked well, and what didn’t, in the sprint just finished
- Propose and agree to changes to improve future sprints
- Assign action items (meetings/tasks) to implement agreed-upon process improvements

Sprint Kick Off (1 hour)  
Who: Development team, product owner.

- Add latest bugs or dependencies to sprint
- Create tasks for each story
- Assign a developer to each task
- Assign an on-call engineer for that sprint who will triage customer support requests
- Check that commitment level is realistic

#### Planning (alternate Wednesdays mid-sprint)

Roadmap review (1 hour)  
Who: Development team, product owner, stakeholders.

- Report high level status of epics
- Prioritize epics
- Define new epics

Sprint Planning (1-2 hours)  
Who: Development team, product owner.

- Discuss and get engineering team consensus on feature design & implementation strategy for tasks on current and upcoming epics
