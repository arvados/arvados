[//]: # Copyright (C) The Arvados Authors. All rights reserved.
[//]: #
[//]: # SPDX-License-Identifier: AGPL-3.0

When run-build-packages.sh builds a Rails package, it generates the package's pre/post-inst/rm scripts by concatenating:

1. package_name.sh, which defines variables about where package files live and some human-readable names about them.
2. step2.sh, which uses those to define some utility variables and set defaults for things that aren't set.
3. stepname.sh, like postinst.sh, prerm.sh, etc., which uses all this information to do the actual work.

Since our build process is a tower of shell scripts, concatenating files seemed like the least worst option to share code between these files and packages.  More advanced code generation would've been too much trouble to integrate into our build process at this time.  Trying to inject portions of files into other files seemed error-prone and likely to introduce bugs to the end result.

postinst.sh lets the early parts define a few hooks to control behavior:

* After it installs the core configuration files (database.yml, application.yml, and production.rb) to /etc/arvados/server, it calls setup_extra_conffiles.  By default this is a noop function (in step2.sh).
* Before it restarts nginx, it calls setup_before_nginx_restart.  By default this is a noop function (in step2.sh).  API server defines this to set up the internal git repository, if necessary.
* $RAILSPKG_DATABASE_LOAD_TASK defines the Rake task to load the database.  API server uses db:structure:load.  Workbench doesn't set this, which causes the postinst to skip all database work.
