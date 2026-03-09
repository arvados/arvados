[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Coding Standards

The rules are always up for debate. However, when debate is needed, it should happen outside the source tree. In other words, if the rules are wrong, first debate the rules at sprint retrospective, then fix the rules, then follow the new rules.

{{toc}}

## Ready to implement checklist

Before starting full implementation, please fill out this template with information about pre-planning:

    * Goals and scope of the ticket are clear to the assigned developer and assigned reviewer.
    ** _comments_
    * New or changed UX/UX has a mockup and has gotten feedback from stakeholders.
    ** _comments_
    * If part of a larger project, ticket is linked to upstream and downstream tasks.
    ** _comments_

UX/UX stands for “User Interface / User Experience”. This includes new or modified GUI elements in workbench and as well as usability elements of command line tools.

Mockups can consist of a wireframe sketched using a drawing tool (e.g. https://excalidraw.com/) or a coding a non-functional prototype focusing on visual design and avoiding implement behavior and uses hardcoded values.

Stakeholders include the rest of the engineering team, as well as designers, salespeople, customers and other end users as appropriate. In this process, the assigned developer presents the mockup, makes note of any feedback, and then based on their judgement either: alters the mockup, iterates on feedback, or begins implementation.

## Ready to merge checklist

Before asking for a branch review, please fill out this template with information about the branch.

<u>template begins below, replace the bits between \< \></u>

    <00000-branch-title> @ commit:<git hash>

    <https://ci.arvados.org/... (link to developer test job on jenkins)>

    _Note each item completed with additional detail if necessary.  If an item is irrelevant to a specific branch, briefly explain why._

    * All agreed upon points are implemented / addressed.  Describe changes from pre-implementation design.
    ** _comments_
    * Anything not implemented (discovered or discussed during work) has a follow-up story.
    ** _comments_
    * Code is tested and passing, both automated and manual, what manual testing was done is described.
    ** _comments_
    * The tested code incorporates recent main branch changes.
    ** _confirm_ ("Incorporates" = merged or rebased. "Recent" = 2-3 working days. The more active development on this component is, the more important it is to be based on recent main to avoid surprising test failures post-merge.)
    * New or changed UI/UX has gotten feedback from stakeholders.
    ** _comments_
    * Documentation has been updated.
    ** _comments_
    * Behaves appropriately at the intended scale (describe intended scale).
    ** _comments_
    * Considered backwards and forwards compatibility issues between client and server.
    ** _comments_
    * Follows our "coding standards":https://dev.arvados.org/projects/arvados/wiki/Coding_Standards and "GUI style guidelines.":https://dev.arvados.org/projects/arvados/wiki/Coding_Standards#GUI-Design-Guidelines-Workbench-2
    ** _comments_

    <Additional detail about what, why and how this branch changes the code>

UI/UX stands for “User Interface / User Experience”. This includes new or modified GUI elements in workbench and as well as usability elements of command line tools.

Stakeholders include the rest of the engineering team, as well as designers, salespeople, customers and other end users as appropriate. In this process, the assigned developer demos the new feature, makes note of any feedback, and then based on their judgement either: implements the changes, provides a reason why the feedback cannot be acted on, or discusses how to handle the feedback with the assigned reviewer and/or product manager.

## Git commits

Make sure your name and email address are correct.

- Use `git config --global user.email foo`example.com@ et al.
- It’s a little unfortunate to have commits with author `foo`myworkstation.local@ but not bad enough to rewrite history, so fix this before you push!

Refer to a story number in the first (summary) line of each commit comment. This first line should be \<80 chars long, and should be followed by a blank line.

- `1234: Remove useless button.`

**When merging/committing to master,** refer to the story number in a way Redmine will notice. Redmine will list these commits/merges on the story page itself.

- `closes #1234`, or
- `refs #1234`, or
- `no issue #` if no Redmine issue is especially relevant.

Use descriptive commit comments.

- Describe the delta between the old and new tree. If possible, describe the delta in **behavior** rather than the source code itself.
- Good: “1234: Support use of spaces in filenames.”
- Good: “1234: Fix crash when user_id is nil.”
- Less good: “Add some controller methods.” (What do they do?)
- Less good: “More progress on UI branch.” (What is different?)
- Less good: “Incorporate Tom’s suggestions.” (Who cares whose suggestions — what changed?)

If further background or explanation is needed, separate it from the summary with a blank line.

- Example: “Users found it confusing that the boxes had different colors even though they represented the same kinds of things.”

**Every commit** (even merge commits) must have a DCO sign-off. See \[\[Developer Certificate Of Origin\]\].

- Example: <code>Arvados-DCO-1.1-Signed-off-by: Joe Smith \<joe.smith@example.com\></code>

Full examples:

    commit 9c6540b9d42adc4a397a28be1ac23f357ba14ab5
    Author: Tom Clegg <tom@curoverse.com>
    Date:   Mon Aug 7 09:58:04 2017 -0400

        12027: Recognize a new "node failed" error message.

        "srun: error: Cannot communicate with node 0.  Aborting job."

        Arvados-DCO-1.1-Signed-off-by: Tom Clegg <tom@curoverse.com>

    commit 0b4800608e6394d66deec9cecea610c5fbbd75ad
    Merge: 6f2ce94 3a356c4
    Author: Tom Clegg <tom@curoverse.com>
    Date:   Thu Aug 17 13:16:36 2017 -0400

        Merge branch '12081-crunch-job-retry'

        refs #12080
        refs #12081
        refs #12108

        Arvados-DCO-1.1-Signed-off-by: Tom Clegg <tom@curoverse.com>

## Copyright headers

Each Arvados component is released either under the AGPL 3.0 license or the Apache 2.0 license. Documentation is licensed under CC-BY-SA-3.0. See the \[\[Arvados Licenses FAQ\]\] for the rationale behind this system.

Every file must start with a copyright header that follows this format:

Code under the [AGPLv3 license](http://www.gnu.org/licenses/agpl-3.0.en.html) (this example uses Go formatting):

    // Copyright (C) The Arvados Authors. All rights reserved.
    //
    // SPDX-License-Identifier: AGPL-3.0

Code under the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0) (this example uses Python formatting):

    # Copyright (C) The Arvados Authors. All rights reserved.
    #
    # SPDX-License-Identifier: Apache-2.0

Documentation under the [Creative Commons Attribution-Share Alike 3.0 United States license](https://creativecommons.org/licenses/by-sa/3.0/us/) (this example uses textile formatting):

    ###. Copyright (C) The Arvados Authors. All rights reserved.
    ....
    .... SPDX-License-Identifier: CC-BY-SA-3.0

When adding a new file to a component, use the same license as the other files of the component.

When adding a new component, choose either the AGPL or Apache license. Generally speaking, the Apache license is only used for components where integrations in proprietary code must be possible (e.g. our SDKs), though this is not a hard rule. When uncertain which license to choose for a new component, ask on the IRC channel or mailing list.

When adding a file in a format that does not support the addition of a copyright header (e.g. in a binary format like an image), add the path to the .licenseignore file in the root of the source tree. This should be done sparingly, and must be discussed explicitly as part of code review. The file must be available under a license that is compatible with the rest of the Arvados code base.

When adding a file that originates from an external source under a different license, add the appropriate SPDX line for that license. This is exceptional, and must be discussed explicitly as part of code review. Not every license is compatible with the rest of the Arvados code base.

There is a helper script at https://github.com/arvados/arvados/blob/master/build/check-copyright-notices that can be used to check - and optionally, fix - the copyright headers in the Arvados source tree.

The actual git hook that enforces the copyright headers lives at https://github.com/arvados/arvados-dev/blob/master/git/hooks/check-copyright-headers.rb

## Source code formatting

(Unless otherwise specified by style guide…)

No TAB characters in source files. [Except go programs.](https://golang.org/cmd/gofmt/)

- Emacs: add to `~/.emacs` → `(setq-default indent-tabs-mode nil)`
- Vim: add to `~/.vimrc` → `:set expandtab`
- See \[\[Coding Standards#Git setup\|Git setup\]\] below

No long (\>80 column) lines, except

- when the alternative is really clunky
- in Go where Google style guide prevails, and e.g., [function and method calls should not be separated based solely on line length](https://google.github.io/styleguide/go/decisions#function-formatting)

No whitespace at the end of lines. Make git-diff show you:

git config color.diff.whitespace “red reverse”  
git diff —check

## What to include

No commented-out blocks of code that have been replaced or obsoleted.

- It is in the git history if we want it back.
- If its absence would confuse someone reading the new code (despite never having read the old code), explain its absence in an English comment. If the old code is really still needed to support the English explanation, then go ahead — now we know why it’s there.

No commented-out debug statements.

- If the debug statements are likely to be needed in the future, use a logging facility that can be enabled at run time. `logger.debug "foo"`

## Style mismatch

Adopt indentation style of surrounding lines or (when starting a new file) the nearest existing source code in this tree/language.

If you fix up existing indentation/formatting, do that in a separate commit.

- If you bundle formatting changes with functional changes, it makes functional changes hard to find in the diff.

## Go

gofmt, golint, etc., and https://github.com/golang/go/wiki/CodeReviewComments

Use `%w` when wrapping an error with fmt.Errorf(), so errors.As() can access the wrapped error.

    <code class="go">
            if err != nil {
                    return fmt.Errorf("could not swap widgets: %w", err)
            }
    </code>

Use `(logrus.FieldLogger)WithError()` (instead of `Logf("blah: %s", err)`) when logging an error.

    <code class="go">
            if err != nil {
                    logger.WithError(err).Warn("error swapping widgets")
            }
    </code>

## Ruby

https://github.com/bbatsov/ruby-style-guide

## Python

### Python code

For code, follow [PEP 8](https://peps.python.org/pep-0008/).

When you add functions, methods, or attributes that SDK users should not use, their name should start with a leading underscore. This is a common convention to signal that an interface is not intended to be public. Anything named this way will be excluded from our SDK web documentation by default.

You’re encouraged to add type annotations to functions and methods. As of May 2024 these are purely for documentation: we are not type checking any of our Python. Note that your annotations must be understood by the oldest version of Python we currently support (3.8).

### Python docstrings

Public classes, methods, and functions should all have docstrings. The content of the docstring should follow [PEP 257](https://peps.python.org/pep-0257/).

Format docstrings with Markdown and follow these style rules:

\* Document function argument lists after the high-level description following this format for each argument:

    * name: type --- Description

Use exactly three minus-hyphens to get an em dash in the web rendering. Provide a helpful type hint whenever practical. The type hint should be written in “modern” style:

Use builtin subscripting from PEP 585/Python 3.9, like `dict[str, str]`, `list[tuple[int, str]]`

Use type union syntax from PEP 604/Python 3.10, like `int | None`, `list[str | bytes]`

Use fully qualified names for custom types. This way pdoc hyperlinks them.

\* When something is deprecated, write a `.. WARNING:: Deprecated` admonition immediately after the first line. Its text should explain that the thing is deprecated, and suggest what to use instead. For example:

    def add(a, b):
        """Add two things.

        .. WARNING:: Deprecated
           This function is deprecated. Use the `+` operator instead.

        …
        """

You can similarly note private methods with `.. ATTENTION:: Internal`.

\* Mark up all identifiers outside the type hint with backticks. When the identifier exists in the current module, use the short name. Otherwise, use the fully-qualified name. Our web documentation will automatically link these identifiers to their corresponding documentation.

\* Mark up links using Markdown’s footnote style. For example:

    """Python docstring following [PEP 257][pep257].


    """

This looks best in plaintext. A descriptive identifier is nice if you can keep it short, but if that’s challenging, plain ordinals are fine too.

\* Mark up headers (e.g., in a module docstring) using underline style. For example:

    """Generic utility module

    Filesystem functions
    --------------------

    …

    Regular expressions
    -------------------

    …
    """

This looks best in plaintext.

The goal of these style rules is to provide a readable, consistent appearance whether people read the documentation in plain text (e.g., using `pydoc`) or their browser (as rendered by `pdoc`).

## JavaScript

Follow the Airbnb Javascript coding style guide unless otherwise stated:  
https://github.com/airbnb/javascript

We already have 4-space indents everywhere, though, so do that.

## Git setup

Configure git to prevent you from committing whitespace errors.

    git config --global core.whitespace tab-in-indent,trailing-space
    git config --global apply.whitespace error

Add a DCO sign-off to the default commit message.

    cd .../arvados
    printf '\n\nArvados-DCO-1.1-Signed-off-by: %s <%s>\n' "$(git config user.name)" "$(git config user.email)" >~/.arvados-dco.txt
    git config commit.template ~/.arvados-dco.txt

Add a DCO sign-off and “refs \#xxxx” comment (referencing the issue# in the name of the branch being merged) to the default merge commit message.

    cd .../arvados
    cat >.git/hooks/prepare-commit-msg <<'EOF'
    #!/bin/sh

    case "$2,$3" in
        merge,)
            br=$(head -n1 ${1})
            n=$(echo "${br}" | egrep -o '[0-9]+')
            exec >${1}
            echo "${br}"
            echo
            echo "refs #${n}"
            echo
            echo "Arvados-DCO-1.1-Signed-off-by: $(git config user.name) <$(git config user.email)>"
            ;;
        *)
            ;;
    esac
    EOF
    chmod +x .git/hooks/prepare-commit-msg

## GUI Design Guidelines (Workbench 2)

### Font Sizes

- Minimum 12pt (16px)
- Minimum 9 pt (12px) for things like by copyright, footer

This should be able to be-resized up to 200% without loss of content or functionality.

### Color

- Text and images of text have a color contrast ratio of at least 4.5:1 You can use [this contrast tool](https://snook.ca/technical/colour_contrast/colour.html#fg=1F7EA1,bg=FFFFFF) to check.
- Non-text icon, controls, etc - 3:1 must have a color contrast ratio of 3:1.
- Avoid hard-coding colors. Use theme colors. If a new color is needed, add it to the theme.
- Used defined grays when possible using RGB value and changing the a value to indicate different meanings (i.e. Active icons have an opacity of 87, Inactive icons have an opacity of 60, Disabled icons have an opacity of 38%)

### Icons

#### General

- Interaction target size of at least 44 x 44 pixels
- Label should be on right, icon on left for maximum readability
- Use minimum 3:1 color contrast (see Color above)
- User appropriate concise alt text for people using screen readers

#### Menu/Navigation

- No navigation should only supported via breadcrumbs
- If less than 5 menu options, consider visible navigation options
- If more than 5 menu options, consider a combination navigation where some options are visible and some are hidden
- Use the following menu consistently:
  - Hamburger (three bars stacked vertically): Used to indicate navigation bar/menu that toggles between being collapsed behind the button or displayed on the screen, often used for global/site-wide/whole application navigation
  - Döner (three bars that narrow vertically): Indicates a group filtering menu
  - Bento (3×3 grid of squares): Indicates a menu presenting a grid of options (not currently applicable to WB)
  - Kebab (three dots stacked vertically): Indicates a smaller inline-menu or an overflow/combination menu
  - Meatballs (three dots stacked horizontally): Used to indicate a smaller inline-menu. Often used to indicate action on a related item (i.e. item next to the meatball), good for repeated use in tables, or horizontal elements
- If component is an accordion window, use caret(‸)

Preferred Icon Repositories:

- https://v5.mui.com/material-ui/material-icons/
- https://materialdesignicons.com/
- https://fontawesome.com/v5/search

### Buttons

- Label button with action for usability/to reduce ambiguity (avoid generic button labels for actions)
- Buttons vs Links
  - Buttons should cause change in current context
  - Links should navigate to a different content or a new resource (e.g. different page)
- If text on button - color contract should be 4.5 :1 between button and text
- Button color and background color contrast should be 3:1

### Arvados Specific Components

Use chips for displaying tokenized values/arrays

### Loading Indicators

#### Page Navigation

- Navigation between pages should be indicated using `progressIndicatorActions.START_WORKING` and `progressIndicatorActions.STOP_WORKING` to show the global top-of-page pulser
- Only the initial load or refresh of the full page (eg. triggered by the upper right refresh button) should use this indicator. Partial refreshes should use a more local indicator.
  - Refreshes of only one section of a page should only show its own loading indicator in that section
- Full page refreshes where the location is unchanged should avoid using the initial full-page spinner in favor of the top-of-page spinner, with updated values substituting in the UI when loaded

#### User Actions

- Form submissions or user actions should be indicated by both the `progressIndicatorActions.START_WORKING` and by enabling the spinner on the submit button of the form (if the action takes place through a form AND if the form stays open for the duration of the action in order to show errors). If the form closes immediately then the page spinner is the only indicator.
- Toasts should not be used to notify the user of an in-progress action but only completion / error

#### Lazy-loaded fields

- Fields that load or update (eg. with extra info) after the main view should wait 3-5 seconds before showing a spinner/pulser icon while loading - if the request for extra data fails, a placeholder icon should show with a hint (text or tooltip) indicating that the data failed to load.
  - The delayed indicator should be implemented as a reusable component (tbd)
- Suggested loading indicator for inline fields: https://mhnpd.github.io/react-loader-spinner/docs/components/three-dots)

### References

[WCAG2.1](https://www.w3.org/WAI/WCAG21/Understanding/)

[Sarah’s talk for references](https://docs.google.com/presentation/d/1HNrhvK7zVZ7jgH3ELbX7KB97SdXCZXrvov_I4Oe1l2c/edit?usp=sharing)
