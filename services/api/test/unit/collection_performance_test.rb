# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/manifest_examples'
require 'helpers/time_block'

class CollectionModelPerformanceTest < ActiveSupport::TestCase
  include ManifestExamples

  setup do
    # The Collection model needs to have a current token, not just a
    # current user, to sign & verify manifests:
    Thread.current[:token] = api_client_authorizations(:active).token
  end

  teardown do
    Thread.current[:token] = nil
  end

  # "crrud" == "create read render update delete", not a typo
  slow_test "crrud cycle for a collection with a big manifest)" do
    bigmanifest = time_block 'make example' do
      make_manifest(streams: 100,
                    files_per_stream: 100,
                    blocks_per_file: 20,
                    bytes_per_block: 2**26,
                    api_token: api_client_authorizations(:active).token)
    end
    act_as_user users(:active) do
      c = time_block "new (manifest_text is #{bigmanifest.length>>20}MiB)" do
        Collection.new manifest_text: bigmanifest.dup
      end
      time_block 'check signatures' do
        c.check_signatures
      end
      time_block 'check signatures + save' do
        c.instance_eval do @signatures_checked = false end
        c.save!
      end
      c = time_block 'read' do
        Collection.find_by_uuid(c.uuid)
      end
      time_block 'render' do
        c.as_api_response(nil)
      end
      loc = Blob.sign_locator(Digest::MD5.hexdigest('foo') + '+3',
                              api_token: api_client_authorizations(:active).token)
      # Note Collection's strip_manifest_text method has now removed
      # the signatures from c.manifest_text, so we have to start from
      # bigmanifest again here instead of just appending with "+=".
      c.manifest_text = bigmanifest.dup + ". #{loc} 0:3:foo.txt\n"
      time_block 'update' do
        c.save!
      end
      time_block 'delete' do
        c.destroy
      end
    end
  end
end
