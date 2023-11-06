# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'

class CollectionUploadTest < ActionDispatch::IntegrationTest
  setup do
    testfiles.each do |filename, content|
      open(testfile_path(filename), 'w') do |io|
        io.write content
      end
    end
    # Database reset doesn't restore KeepServices; we have to
    # save/restore manually.
    use_token :admin do
      @keep_services = KeepService.all.to_a
    end
  end

  teardown do
    use_token :admin do
      @keep_services.each do |ks|
        KeepService.find(ks.uuid).update(ks.attributes)
      end
    end
    testfiles.each do |filename, _|
      File.unlink(testfile_path filename)
    end
  end

  test "Create new collection using upload button" do
    need_javascript
    visit page_with_token 'active', aproject_path
    find('.btn', text: 'Add data').click
    click_link 'Upload files from my computer'
    # Should be looking at a new empty collection.
    assert_text 'New collection'
    assert_text ' 0 files'
    assert_text ' 0 bytes'
    # The "Upload" tab should be active and loaded.
    assert_selector 'div#Upload.active div.panel'
  end

  test "Upload two empty files with the same name" do
    need_selenium "to make file uploads work"
    visit page_with_token 'active', sandbox_path

    unlock_collection

    find('.nav-tabs a', text: 'Upload').click
    attach_file 'file_selector', testfile_path('empty.txt')
    assert_selector 'div', text: 'empty.txt'
    attach_file 'file_selector', testfile_path('empty.txt')
    assert_selector 'div.row div span[title]', text: 'empty.txt', count: 2
    click_button 'Start'
    assert_text :visible, 'Done!'
    visit sandbox_path+'.json'
    assert_match /_text":"\. d41d8\S+ 0:0:empty.txt\\n\. d41d8\S+ 0:0:empty\\\\040\(1\).txt\\n"/, body
  end

  test "Upload non-empty files" do
    need_selenium "to make file uploads work"
    visit page_with_token 'active', sandbox_path

    unlock_collection

    find('.nav-tabs a', text: 'Upload').click
    attach_file 'file_selector', testfile_path('a')
    attach_file 'file_selector', testfile_path('foo.txt')
    assert_selector 'button:not([disabled])', text: 'Start'
    click_button 'Start'
    assert_text :visible, 'Done!'
    visit sandbox_path+'.json'
    assert_match /_text":"\. 0cc1\S+ 0:1:a\\n\. acbd\S+ 0:3:foo.txt\\n"/, body
  end

  test "Report mixed-content error" do
    skip 'Test suite does not use TLS'
    need_selenium "to make file uploads work"
    use_token :admin do
      KeepService.where(service_type: 'proxy').first.
        update(service_ssl_flag: false)
    end
    visit page_with_token 'active', sandbox_path
    find('.nav-tabs a', text: 'Upload').click
    attach_file 'file_selector', testfile_path('foo.txt')
    assert_selector 'button:not([disabled])', text: 'Start'
    click_button 'Start'
    using_wait_time 5 do
      assert_text :visible, 'server setup problem'
      assert_text :visible, 'cannot be used from origin'
    end
  end

  test "Report network error" do
    need_selenium "to make file uploads work"
    use_token :admin do
      # Even if port 0 is a thing, surely nx.example.net won't
      # respond
      KeepService.where(service_type: 'proxy').first.
        update(service_host: 'nx.example.net',
                          service_port: 0)
    end
    visit page_with_token 'active', sandbox_path

    unlock_collection

    find('.nav-tabs a', text: 'Upload').click
    attach_file 'file_selector', testfile_path('foo.txt')
    assert_selector 'button:not([disabled])', text: 'Start'
    click_button 'Start'
    using_wait_time 5 do
      assert_text :visible, 'network error'
    end
  end

  protected

  def aproject_path
    '/projects/' + api_fixture('groups')['aproject']['uuid']
  end

  def sandbox_uuid
    api_fixture('collections')['upload_sandbox']['uuid']
  end

  def sandbox_path
    '/collections/' + sandbox_uuid
  end

  def testfiles
    {
      'empty.txt' => '',
      'a' => 'a',
      'foo.txt' => 'foo'
    }
  end

  def testfile_path filename
    # Must be an absolute path. https://github.com/jnicklas/capybara/issues/621
    File.join Dir.getwd, 'tmp', filename
  end

  def unlock_collection
    first('.lock-collection-btn').click
    accept_alert
  end
end
