#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

exec docker build -t arvados/migrate-docker19:1.0 .
