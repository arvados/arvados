# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SaferFileStore < ActiveSupport::Cache::FileStore
  private
  def delete_empty_directories(dir)
    # It is not safe to delete an empty directory. Another thread or
    # process might be in write_entry(), having just created an empty
    # directory via ensure_cache_path(). If we delete that empty
    # directory, the other thread/process will crash in
    # File.atomic_write():
    #
    # #<Errno::ENOENT: No such file or directory @ rb_sysopen - /.../tmp/cache/94F/070/.permissions_check.13730420.54542.801783>
  end
end
