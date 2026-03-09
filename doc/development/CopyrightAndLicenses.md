[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Copyright and licenses

Every commit (even merge commits) must have a DCO sign-off. See \[\[Developer Certificate Of Origin\]\].

Most source files must have a copyright notice and license statement.

- Example: source:services/api/Gemfile
- Run `build/check-copyright-notices` to check
- Run `build/check-copyright-notices --fix` to add the appropriate statement to files where it’s missing (please preview result before committing!)
- Run `build/check-copyright-notices --fix -- path/to/file1 path/to/file2 ...` to check only specified file(s) (much faster!)

## Third-party code in tree

It is acceptable to copy third-party code into the source tree, although there’s usually a better way (e.g., use a Ruby gem and let bundler pull the code from an external repository at package-building time).

Third-party code that has been copied into the source tree:

- list.js, MIT (source:apps/workbench/app/assets/javascripts/list.js)
- sb-admin, Apache2 (source:apps/workbench/app/assets/stylesheets/sb-admin.css.scss)
- shell_in_a_box, GPLv2 (source:apps/workbench/public/webshell/)
- jquery.number.js, MIT (source:apps/workbench/vendor/assets/javascripts/jquery.number.min.js)
- bootstrap+fontawesome+glyphicons, MIT (source:doc/css/bootstrap.css, source:doc/fonts/, etc)
- runit-docker, BSD 3-clause (source:tools/arvbox/lib/arvbox/docker/runit-docker/)
