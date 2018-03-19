# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module ContainerTestHelper
  def secret_string
    'UNGU3554BL3'
  end

  def assert_no_secrets_logged
    Log.all.map(&:properties).each do |props|
      refute_match /secret\/6x9|#{secret_string}/, SafeJSON.dump(props)
    end
  end
end
