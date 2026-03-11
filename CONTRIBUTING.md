[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Contributing to Arvados

Arvados is free software, which means it is free for all to use, learn
from, and improve.  We encourage contributions from the community that
improve Arvados for everyone.  Some examples of contributions are bug
reports, bug fixes, new features, and scripts or documentation that help
with using, administering, or installing Arvados.  We also love to
hear about Arvados success stories.

## Reporting Issues

Arvados uses [GitHub Issues](https://github.com/arvados/arvados/issues). You can file issues against any Arvados component there. Even if you're not sure which component causes the issue, you can still file problem reports and we'll work with you to address them.

## Contributing Code

The preferred method for making contributions is through GitHub pull requests. The rest of this guide helps orient you with the code and discusses requirements for all contributions, from the smallest typo fix to entire new components.

If you're interested in developing a large new feature for Arvados, please file an issue to discuss it with us first. We can give you guidance on how to best organize the work before you start it.

## Setting Up Your Development Environment

The [Arvados source code is hosted on GitHub](https://github.com/arvados/arvados). Once you clone it, you'll find guides for specific topics under the `doc/development` directory. You'll probably want to [install a development environment](doc/development/Prerequisites.md) and [learn how to run tests](doc/development/RunningTests.md). There are also some component-specific guides.

### Setting Up Git

We provide Git configuration and hooks to help you follow project conventions.

`doc/development/git.conf` includes a block of Git configuration settings. You can set it up for your checkout by running `git config edit --local`: insert the contents of `doc/development/git.conf`, edit them following the comments, then save and exit.

Install our `prepare-commit-msg` hook:

```sh
$ install -b -m 755 doc/development/prepare-commit-msg.sh .git/hooks/prepare-commit-msg
```

## Prepare a Development Branch

If you haven't before, fork the Arvados repository using the GitHub "Fork" button. If you have, make sure your fork's `main` branch is up-to-date with Arvados'.

Then start a new branch for your development named like `1234-your-work`. The number at the start should match the GitHub issue this request is associated with. Then briefly describe the main change your branch makes.

### Coding Standards

Please familiarize yourself with our [coding standards](doc/development/CodingStandards.md) for the component(s) you're working on and follow them in your work.

### Sign Off Your Commits

Contributions must be signed off. The sign-off is a simple line at the end of each commit message  which certifies that you wrote it or otherwise have the right to contribute it under the license listed in the file(s) modified. Make sure each commit message contains the following line with your real name and email (sorry, no pseudonymous or anonymous contributions):

    Arvados-DCO-1.1-Signed-off-by: Alex Doe <alex.doe@example.com>

When you add this, you certify the below (from <https://developercertificate.org>):

> Developer Certificate of Origin  
> Version 1.1
>
> Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
>
> Everyone is permitted to copy and distribute verbatim copies of this
> license document, but changing it is not allowed.
>
>
> Developer's Certificate of Origin 1.1
>
> By making a contribution to this project, I certify that:
>
> (a) The contribution was created in whole or in part by me and I
>     have the right to submit it under the open source license
>     indicated in the file; or
>
> (b) The contribution is based upon previous work that, to the best
>     of my knowledge, is covered under an appropriate open source
>     license and I have the right under that license to submit that
>     work with modifications, whether created in whole or in part
>     by me, under the same open source license (unless I am
>     permitted to submit under a different license), as indicated
>     in the file; or
>
> (c) The contribution was provided directly to me by some other
>     person who certified (a), (b) or (c) and I have not modified
>     it.
>
> (d) I understand and agree that this project and the contribution
>     are public and that a record of the contribution (including all
>     personal information I submit with it, including my sign-off) is
>     maintained indefinitely and may be redistributed consistent with
>     this project or the open source license(s) involved.

### Add License Headers

The comments at the top of each file must contain this copyright notice:

> Copyright © The Arvados Authors. All rights reserved.

They must also contain an `SPDX-License-Identifier` to identify the license of this component.

In most cases you can copy this header from another file in the component. If you need more guidance, refer to [the COPYING file](COPYING).

If it is not technically possible to add these comments to a file (for example, because it's a binary test file), you may add its path to the `.licenseignore` file instead.

### Add Your Authorship

If you are not already listed in [the AUTHORS file](AUTHORS), please add yourself in the branch, following the documented format.

## Create Your Pull Request

Once you've finished pushing changes to your branch, create a pull request against `arvados:main` with the following checklist filled out:

    * All agreed upon points are implemented / addressed.  Describe changes from pre-implementation design.
    ** _comments_
    * Anything not implemented (discovered or discussed during work) has a follow-up story.
    ** _comments_
    * Code is tested and passing, both automated and manual, what manual testing was done is described.
    ** _comments_
    * The tested code incorporates recent main branch changes.
    ** _confirm_
    * New or changed UI/UX has gotten feedback from stakeholders.
    ** _comments_
    * Documentation has been updated.
    ** _comments_
    * Behaves appropriately at the intended scale (describe intended scale).
    ** _comments_
    * Considered backwards and forwards compatibility issues between client and server.
    ** _comments_
    * Follows our coding standards, including GUI style guidelines
    ** _comments_

"Incorporates recent main branch changes" means that the branch is either based on, or merged, the `main` branch within the last week. The more active development on a component is, the more important it is to be up-to-date with main to avoid surprising test failures post-merge.

UI/UX stands for “User Interface / User Experience”. This includes new or modified GUI elements in Workbench and as well as usability elements of command line tools.

Stakeholders typically include the product manager and may include designers, salespeople, customers, and other end users as appropriate. In this process, the assigned developer demos the new feature, makes note of any feedback, and then based on their judgement either: implements the changes, provides a reason why the feedback cannot be acted on, or discusses how to handle the feedback with the product manager and/or assigned reviewer. This feedback is typically obtained in earlier drafts of the pull request before it is submitted for final review.

A member of the core team will review the pull request. They may have questions or comments through the pull request interface. Once all issues have been resolved, your branch will be merged.

## Continuous Integration

Continuous integration is hosted at <https://ci.arvados.org/>. Currently, external contributors cannot trigger test runs. Trusted contributors may be given permission to do so.

## Community Chat

You can chat with other members of the [Arvados community on Gitter](https://gitter.im/arvados/community). Come say hi!
