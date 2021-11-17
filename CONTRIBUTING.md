[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Contributing

Arvados is free software, which means it is free for all to use, learn
from, and improve.  We encourage contributions from the community that
improve Arvados for everyone.  Some examples of contributions are bug
reports, bug fixes, new features, and scripts or documentation that help
with using, administering, or installing Arvados.  We also love to
hear about Arvados success stories.

Those interested in contributing should begin by joining the [Arvados community
channel](https://gitter.im/arvados/community) and telling us about your interest.

Contributors should also create an account at https://dev.arvados.org
to be able to create and comment on bug tracker issues.  The
Arvados public bug tracker is located at
https://dev.arvados.org/projects/arvados/issues .

Contributors may also be interested in the [development road map](https://dev.arvados.org/issues/gantt?utf8=%E2%9C%93&set_filter=1&gantt=1&f%5B%5D=project_id&op%5Bproject_id%5D=%3D&v%5Bproject_id%5D%5B%5D=49&f%5B%5D=&zoom=1).

# Development

Git repositories for primary development are located at
https://git.arvados.org/ and can also be browsed at
https://dev.arvados.org/projects/arvados/repository .  Every push to
the main branch is also mirrored to Github at
https://github.com/arvados/arvados .

Visit [Hacking Arvados](https://dev.arvados.org/projects/arvados/wiki/Hacking) for
detailed information about setting up an Arvados development
environment, development process, [coding standards](https://dev.arvados.org/projects/arvados/wiki/Coding_Standards), and notes about specific components.

If you wish to build the Arvados documentation from a local git clone, see
[doc/README.textile](doc/README.textile) for instructions.

# Pull requests

The preferred method for making contributions is through Github pull requests.

This is the general contribution process:

1. Fork the Arvados repository using the Github "Fork" button
2. Clone your fork, make your changes, commit to your fork.
3. Every commit message must have a DCO sign-off and every file must have a SPDX license (see below).
4. Add yourself to the [AUTHORS](AUTHORS) file
5. When your fork is ready, through Github, Create a Pull Request against `arvados:main`
6. Notify the core team about your pull request through the [Arvados development
channel](https://gitter.im/arvados/development) or by other means.
7. A member of the core team will review the pull request.  They may have questions or comments, or request changes.
8. When the contribution is ready, a member of the core team will
merge the pull request into the main branch, which will
automatically resolve the pull request.

The Arvados project does not require a contributor agreement in advance, but does require each commit message include a [Developer Certificate of Origin](https://dev.arvados.org/projects/arvados/wiki/Developer_Certificate_Of_Origin).  Please ensure *every git commit message* includes `Arvados-DCO-1.1-Signed-off-by`. If you have already made commits without it, fix them with `git commit --amend` or `git rebase`.

The Developer Certificate of Origin line looks like this:

```
Arvados-DCO-1.1-Signed-off-by: Joe Smith <joe.smith@example.com>
```

New files must also include `SPDX-License-Identifier` at the top with one of the three Arvados open source licenses.  See [COPYING](COPYING) for details.

# Continuous integration

Continuous integration is hosted at https://ci.arvados.org/

Currently, external contributors cannot trigger builds.  We are investigating integration with Github pull requests for the future.

[![Build Status](https://ci.arvados.org/buildStatus/icon?job=run-tests)](https://ci.arvados.org/job/run-tests/)

[![Go Report Card](https://goreportcard.com/badge/github.com/arvados/arvados)](https://goreportcard.com/report/github.com/arvados/arvados)
