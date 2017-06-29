# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module DockerMigrationHelper
  include CurrentApiClient

  def add_docker19_migration_link
    act_as_system_user do
      assert(Link.create!(owner_uuid: system_user_uuid,
                          link_class: 'docker_image_migration',
                          name: 'migrate_1.9_1.10',
                          tail_uuid: collections(:docker_image).portable_data_hash,
                          head_uuid: collections(:docker_image_1_12).portable_data_hash))
    end
  end
end
