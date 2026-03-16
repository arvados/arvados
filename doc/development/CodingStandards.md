[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Coding Standards

The rules are always up for debate. However, when debate is needed, it should happen outside the source tree. In other words, if the rules are wrong, first debate the rules at sprint retrospective, then fix the rules, then follow the new rules.

## Git commit messages

- Prefix the summary line with the issue number this addresses.
- Describe the delta between the old and new tree. If possible, describe the delta in **behavior** rather than the source code itself.
- Good: “1234: Support use of spaces in filenames.”
- Good: “1234: Fix crash when user_id is nil.”
- Less good: “Add some controller methods.” (What do they do?)
- Less good: “More progress on UI branch.” (What is different?)
- Less good: “Incorporate Tom’s suggestions.” (Who cares whose suggestions — what changed?)

If further background or explanation is needed, separate it from the summary with a blank line.

- Example: “Users found it confusing that the boxes had different colors even though they represented the same kinds of things.”

**Every commit** (including merge commits) must have a DCO sign-off. See `CONTRIBUTING.md` for the full terms of what this means.

- Example: `Arvados-DCO-1.1-Signed-off-by: Alex Doe <alex.doe@example.com>`

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

## Source code formatting

These are general baseline rules except when a language-specific guide specifies otherwise.

No TAB characters in source files [except Go](https://golang.org/cmd/gofmt/).

- For Emacs, add `(setq-default indent-tabs-mode nil)` to `~/.emacs`.
- For Vim, add `:set expandtab` to `~/.vimrc`.

Avoid long (\>100 column) lines.

No whitespace at the end of lines unless technically required (like Markdown line breaks).

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

Follow gofmt, golint, etc., and <https://github.com/golang/go/wiki/CodeReviewComments>

Use `%w` when wrapping an error with fmt.Errorf(), so errors.As() can access the wrapped error.

```go
if err != nil {
        return fmt.Errorf("could not swap widgets: %w", err)
}
```

Use `(logrus.FieldLogger)WithError()` (instead of `Logf("blah: %s", err)`) when logging an error.

```go
if err != nil {
        logger.WithError(err).Warn("error swapping widgets")
}
```

## Ruby

Follow <https://github.com/bbatsov/ruby-style-guide>

## Python

### Python code

For code, follow [PEP 8](https://peps.python.org/pep-0008/).

When you add functions, methods, or attributes that SDK users should not use, their name should start with a leading underscore. This is a common convention to signal that an interface is not intended to be public. Anything named this way will be excluded from our SDK web documentation by default.

You’re encouraged to add type annotations to functions and methods. As of May 2024 these are purely for documentation: we are not type checking any of our Python. Note that your annotations must be understood by the oldest version of Python we currently support (3.10).

### Python docstrings

Public classes, methods, and functions should all have docstrings. The content of the docstring should follow [PEP 257](https://peps.python.org/pep-0257/).

Format docstrings with Markdown and follow these style rules:

* Document function argument lists after the high-level description following this format for each argument:

        * name: type --- Description

   Use exactly three minus-hyphens to get an em dash in the web rendering. Provide a helpful type hint whenever practical. The type hint should be written in “modern” style with builtin subscripting and type union syntax, like `list[str | bytes]`.

   Use fully qualified names for custom types. This way pdoc hyperlinks them.

* When something is deprecated, write a `.. WARNING:: Deprecated` admonition immediately after the first line. Its text should explain that the thing is deprecated, and suggest what to use instead. For example:

        def add(a, b):
            """Add two things.

            .. WARNING:: Deprecated
               This function is deprecated. Use the `+` operator instead.

            …
            """

   You can similarly note private methods with `.. ATTENTION:: Internal`.

* Mark up all identifiers outside the type hint with backticks. When the identifier exists in the current module, use the short name. Otherwise, use the fully-qualified name. Our web documentation will automatically link these identifiers to their corresponding documentation.

* Mark up links using Markdown’s footnote style. For example:

        """Python docstring following [PEP 257][pep257].


        """

   This looks best in plaintext. A descriptive identifier is nice if you can keep it short, but if that’s challenging, plain ordinals are fine too.

* Mark up headers (e.g., in a module docstring) using underline style. For example:

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

We already have 4-space indents everywhere, so do that.

Other than that, follow the [Airbnb Javascript coding style](https://github.com/airbnb/javascript) guide unless otherwise stated.

## Workbench Design Guidelines

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

- <https://v5.mui.com/material-ui/material-icons/>
- <https://materialdesignicons.com/>
- <https://fontawesome.com/v5/search>

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
