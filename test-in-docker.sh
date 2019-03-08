#!/bin/sh
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
#

set -e
UID=$(id -u)
docker run --user $UID -v $PWD:$PWD -w $PWD java:8 /bin/sh -c '(./gradlew clean && ./gradlew test); ./gradlew --stop'
