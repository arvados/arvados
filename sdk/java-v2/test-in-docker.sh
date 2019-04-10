#!/bin/sh
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
#

set -e
#UID=$(id -u) # UID is read-only on many systems
exec docker run --rm --user $UID -v $PWD:$PWD -w $PWD gradle /bin/sh -c '(gradle clean && gradle test); gradle --stop'
