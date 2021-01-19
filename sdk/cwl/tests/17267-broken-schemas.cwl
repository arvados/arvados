# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
$schemas:
  - http://example.com/schema.xml
inputs: []
outputs:
  out: stdout
baseCommand: [echo, "foo"]
stdout: foo.txt
