# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module ArvadosModelUpdates
  # ArvadosModel checks this to decide whether it should update the
  # 'modified_by_user_uuid' field.
  def anonymous_updater
    Thread.current[:anonymous_updater] || false
  end

  def leave_modified_by_user_alone
    anonymous_updater_was = anonymous_updater
    begin
      Thread.current[:anonymous_updater] = true
      yield
    ensure
      Thread.current[:anonymous_updater] = anonymous_updater_was
    end
  end

  # ArvadosModel checks this to decide whether it should update the
  # 'modified_at' field.
  def timeless_updater
    Thread.current[:timeless_updater] || false
  end

  def leave_modified_at_alone
    timeless_updater_was = timeless_updater
    begin
      Thread.current[:timeless_updater] = true
      yield
    ensure
      Thread.current[:timeless_updater] = timeless_updater_was
    end
  end

end
