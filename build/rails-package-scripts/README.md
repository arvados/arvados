[//]: # Copyright (C) The Arvados Authors. All rights reserved.
[//]: #
[//]: # SPDX-License-Identifier: AGPL-3.0

When run-build-packages.sh builds a Rails package, it generates the package's pre/post-inst/rm scripts by concatenating `arvados-api-server.sh` to define common variables, then the actual step script. Especially when this infrastructure was shared with the old Rails Workbench, this seemed like the least worst option to share code between these files and packages.  More advanced code generation would've been too much trouble to integrate into our build process at this time.  Trying to inject portions of files into other files seemed error-prone and likely to introduce bugs to the end result.
