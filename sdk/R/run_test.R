# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

devtools::check()

results <- devtools::test()
any_error <- any(as.data.frame(results)$error)
if (any_error) {
  q("no", 1)
} else {
  q("no", 0)
}
