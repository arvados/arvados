# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Fulltext search indexes were removed in
# 7f4d69cf43a7a743a491105665b3b878a3cfd11c (#15430), but then for no
# apparent reason dcdf385b2852acf95f41e2340d07cd68cb34e371 (#12430)
# re-added the FTS index for container_requests.
class DropFtsIndexAgain < ActiveRecord::Migration[7.0]
  def up
    execute "DROP INDEX IF EXISTS container_requests_full_text_search_idx"
  end

  def down
    # No-op because the index was not used by prior versions either.
  end
end
