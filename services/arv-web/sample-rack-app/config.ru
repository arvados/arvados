# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

app = proc do |env|
    [200, { "Content-Type" => "text/html" }, ["hello <b>world</b> from ruby"]]
end
run app
