# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'
require_relative 'integration_test_utils'

class CollectionsTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  test "Can copy a collection to a project" do
    collection_uuid = api_fixture('collections')['foo_file']['uuid']
    collection_name = api_fixture('collections')['foo_file']['name']
    project_uuid = api_fixture('groups')['aproject']['uuid']
    project_name = api_fixture('groups')['aproject']['name']
    visit page_with_token('active', "/collections/#{collection_uuid}")
    click_link 'Copy to project...'
    find('.selectable', text: project_name).click
    find('.modal-footer a,button', text: 'Copy').click
    # Should navigate to the Data collections tab of the project after copying
    assert_text project_name
    assert_text "Copy of #{collection_name}"
  end

  def check_sharing(want_state, link_regexp)
    # We specifically want to click buttons.  See #4291.
    if want_state == :off
      click_button "Unshare"
      text_assertion = :assert_no_text
      link_assertion = :assert_empty
    else
      click_button "Create sharing link"
      text_assertion = :assert_text
      link_assertion = :refute_empty
    end
    using_wait_time(Capybara.default_max_wait_time * 3) do
      send(text_assertion, "Shared at:")
    end
    send(link_assertion, all("a").select { |a| a[:href] =~ link_regexp })
  end

  test "Hides sharing link button when configured to do so" do
    Rails.configuration.Workbench.DisableSharingURLsUI = true
    coll_uuid = api_fixture("collections", "collection_owned_by_active", "uuid")
    visit page_with_token("active_trustedclient", "/collections/#{coll_uuid}")
    assert_no_selector 'input', text: 'Create sharing link'
  end

  test "creating and uncreating a sharing link" do
    coll_uuid = api_fixture("collections", "collection_owned_by_active", "uuid")
    download_link_re =
      Regexp.new(Regexp.escape("/c=#{coll_uuid}/"))
    visit page_with_token("active_trustedclient", "/collections/#{coll_uuid}")
    assert_selector 'input', text: 'Create sharing link'
    within "#sharing-button" do
      check_sharing(:on, download_link_re)
      check_sharing(:off, download_link_re)
    end
  end

  test "can download an entire collection with a reader token" do
    need_selenium "phantomjs does not follow redirects reliably, maybe https://github.com/ariya/phantomjs/issues/10389"

    token = api_token('active')
    data = "foo\nfile\n"
    datablock = `echo -n #{data.shellescape} | ARVADOS_API_TOKEN=#{token.shellescape} arv-put --no-progress --raw -`.strip
    assert $?.success?, $?

    col = nil
    use_token 'active' do
      mtxt = ". #{datablock} 0:#{data.length}:foo\n"
      col = Collection.create(manifest_text: mtxt)
    end

    uuid = col.uuid
    token = api_fixture('api_client_authorizations')['active_all_collections']['api_token']
    url_head = "/collections/download/#{uuid}/#{token}/"
    visit url_head
    assert_text "You can download individual files listed below"
    # It seems that Capybara can't inspect tags outside the body, so this is
    # a very blunt approach.
    assert_no_match(/<\s*meta[^>]+\bnofollow\b/i, page.html,
                    "wget prohibited from recursing the collection page")
    # Look at all the links that wget would recurse through using our
    # recommended options, and check that it's exactly the file list.
    hrefs = []
    page.html.scan(/href="(.*?)"/) { |m| hrefs << m[0] }
    assert_equal(['./foo'], hrefs, "download page did provide strictly file links")
    click_link "foo"
    assert_text "foo\nfile\n"
  end

  test "combine selected collections into new collection" do
    foo_collection = api_fixture('collections')['foo_file']
    bar_collection = api_fixture('collections')['bar_file']

    visit page_with_token('active', "/collections")

    assert(page.has_text?(foo_collection['uuid']), "Collection page did not include foo file")
    assert(page.has_text?(bar_collection['uuid']), "Collection page did not include bar file")

    within "tr[data-object-uuid=\"#{foo_collection['uuid']}\"]" do
      find('input[type=checkbox]').click
    end

    within "tr[data-object-uuid=\"#{bar_collection['uuid']}\"]" do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Create new collection with selected collections'
    end

    # now in the newly created collection page
    assert(page.has_text?('Copy to project'), "Copy to project text not found in new collection page")
    assert(page.has_no_text?(foo_collection['name']), "Collection page did not include foo file")
    assert(page.has_text?('foo'), "Collection page did not include foo file")
    assert(page.has_no_text?(bar_collection['name']), "Collection page did not include foo file")
    assert(page.has_text?('bar'), "Collection page did not include bar file")
    assert(page.has_text?('Created new collection in your Home project'),
                          'Not found flash message that new collection is created in Home project')
  end

  [
    ['active', 'foo_file', false],
    ['active', 'foo_collection_in_aproject', true],
    ['project_viewer', 'foo_file', false],
    ['project_viewer', 'foo_collection_in_aproject', false], #aproject not writable
  ].each do |user, collection, expect_collection_in_aproject|
    test "combine selected collection files into new collection #{user} #{collection} #{expect_collection_in_aproject}" do
      my_collection = api_fixture('collections')[collection]

      visit page_with_token(user, "/collections")

      # choose file from foo collection
      within('tr', text: my_collection['uuid']) do
        click_link 'Show'
      end

      # now in collection page
      find('input[type=checkbox]').click

      click_button 'Selection...'
      within('.selection-action-container') do
        click_link 'Create new collection with selected files'
      end

      # now in the newly created collection page
      assert(page.has_text?('Copy to project'), "Copy to project text not found in new collection page")
      assert(page.has_no_text?(my_collection['name']), "Collection page did not include foo file")
      assert(page.has_text?('foo'), "Collection page did not include foo file")
      if expect_collection_in_aproject
        aproject = api_fixture('groups')['aproject']
        assert page.has_text?("Created new collection in the project #{aproject['name']}"),
                              'Not found flash message that new collection is created in aproject'
      else
        assert page.has_text?("Created new collection in your Home project"),
                              'Not found flash message that new collection is created in Home project'
      end
    end
  end

  test "combine selected collection files from collection subdirectory" do
    visit page_with_token('user1_with_load', "/collections/zzzzz-4zz18-filesinsubdir00")

    # now in collection page
    input_files = page.all('input[type=checkbox]')
    (0..input_files.count-1).each do |i|
      input_files[i].click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Create new collection with selected files'
    end

    # now in the newly created collection page
    assert(page.has_text?('file_in_subdir1'), 'file not found - file_in_subdir1')
    assert(page.has_text?('file1_in_subdir3.txt'), 'file not found - file1_in_subdir3.txt')
    assert(page.has_text?('file2_in_subdir3.txt'), 'file not found - file2_in_subdir3.txt')
    assert(page.has_text?('file1_in_subdir4.txt'), 'file not found - file1_in_subdir4.txt')
    assert(page.has_text?('file2_in_subdir4.txt'), 'file not found - file1_in_subdir4.txt')
  end

  test "Collection portable data hash with multiple matches with more than one page of results" do
    pdh = api_fixture('collections')['baz_file']['portable_data_hash']
    visit page_with_token('admin', "/collections/#{pdh}")

    assert_selector 'a', text: 'Collection_1'

    assert_text 'The following collections have this content:'
    assert_text 'more results are not shown'
    assert_no_text 'Activity'
    assert_no_text 'Sharing and permissions'
  end

  test "Filtering collection files by regexp" do
    col = api_fixture('collections', 'multilevel_collection_1')
    visit page_with_token('active', "/collections/#{col['uuid']}")

    # Filter file list to some but not all files in the collection
    page.find_field('file_regex').set('file[12]')
    assert page.has_text?("file1")
    assert page.has_text?("file2")
    assert page.has_no_text?("file3")

    # Filter file list with a regex matching all files
    page.find_field('file_regex').set('.*')
    assert page.has_text?("file1")
    assert page.has_text?("file2")
    assert page.has_text?("file3")

    # Filter file list to a regex matching no files
    page.find_field('file_regex').set('file9')
    assert page.has_no_text?("file1")
    assert page.has_no_text?("file2")
    assert page.has_no_text?("file3")
    # make sure that we actually are looking at the collections
    # page and not e.g. a fiddlesticks
    assert page.has_text?("multilevel_collection_1")
    assert page.has_text?(col["name"] || col["uuid"])

    # Set filename filter to a syntactically invalid regex
    # Page loads, but stops filtering after the last valid regex parse
    page.find_field('file_regex').set('file[2')
    assert page.has_text?("multilevel_collection_1")
    assert page.has_text?(col["name"] || col["uuid"])
    assert page.has_text?("file1")
    assert page.has_text?("file2")
    assert page.has_text?("file3")

    # Test the "Select all" button

    # Note: calling .set('') on a Selenium element is not sufficient
    # to reset the field for this test, as it does not send any key
    # events to the browser. To clear the field, we must instead send
    # a backspace character.
    # See https://selenium.googlecode.com/svn/trunk/docs/api/rb/Selenium/WebDriver/Element.html#clear-instance_method
    page.find_field('file_regex').set("\b") # backspace
    find('button#select-all').click
    assert_checkboxes_state('input[type=checkbox]', true, '"select all" should check all checkboxes')

    # Test the "Unselect all" button
    page.find_field('file_regex').set("\b") # backspace
    find('button#unselect-all').click
    assert_checkboxes_state('input[type=checkbox]', false, '"unselect all" should clear all checkboxes')

    # Filter files, then "select all", then unfilter
    page.find_field('file_regex').set("\b") # backspace
    find('button#unselect-all').click
    page.find_field('file_regex').set('file[12]')
    find('button#select-all').click
    page.find_field('file_regex').set("\b") # backspace

    # all "file1" and "file2" checkboxes must be selected
    # all "file3" checkboxes must be clear
    assert_checkboxes_state('[value*="file1"]', true, 'checkboxes for file1 should be selected after filtering')
    assert_checkboxes_state('[value*="file2"]', true, 'checkboxes for file2 should be selected after filtering')
    assert_checkboxes_state('[value*="file3"]', false, 'checkboxes for file3 should be clear after filtering')

    # Select all files, then filter, then "unselect all", then unfilter
    page.find_field('file_regex').set("\b") # backspace
    find('button#select-all').click
    page.find_field('file_regex').set('file[12]')
    find('button#unselect-all').click
    page.find_field('file_regex').set("\b") # backspace

    # all "file1" and "file2" checkboxes must be clear
    # all "file3" checkboxes must be selected
    assert_checkboxes_state('[value*="file1"]', false, 'checkboxes for file1 should be clear after filtering')
    assert_checkboxes_state('[value*="file2"]', false, 'checkboxes for file2 should be clear after filtering')
    assert_checkboxes_state('[value*="file3"]', true, 'checkboxes for file3 should be selected after filtering')
  end

  test "Creating collection from list of filtered files" do
    col = api_fixture('collections', 'collection_with_files_in_subdir')
    visit page_with_token('user1_with_load', "/collections/#{col['uuid']}")
    assert page.has_text?('file_in_subdir1'), 'expected file_in_subdir1 not found'
    assert page.has_text?('file1_in_subdir3'), 'expected file1_in_subdir3 not found'
    assert page.has_text?('file2_in_subdir3'), 'expected file2_in_subdir3 not found'
    assert page.has_text?('file1_in_subdir4'), 'expected file1_in_subdir4 not found'
    assert page.has_text?('file2_in_subdir4'), 'expected file2_in_subdir4 not found'

    # Select all files but then filter them to files in subdir1, subdir2 or subdir3
    find('button#select-all').click
    page.find_field('file_regex').set('_in_subdir[123]')
    assert page.has_text?('file_in_subdir1'), 'expected file_in_subdir1 not in filtered files'
    assert page.has_text?('file1_in_subdir3'), 'expected file1_in_subdir3 not in filtered files'
    assert page.has_text?('file2_in_subdir3'), 'expected file2_in_subdir3 not in filtered files'
    assert page.has_no_text?('file1_in_subdir4'), 'file1_in_subdir4 found in filtered files'
    assert page.has_no_text?('file2_in_subdir4'), 'file2_in_subdir4 found in filtered files'

    # Create a new collection
    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Create new collection with selected files'
    end

    # now in the newly created collection page
    # must have files in subdir1 and subdir3 but not subdir4
    assert page.has_text?('file_in_subdir1'), 'file_in_subdir1 missing from new collection'
    assert page.has_text?('file1_in_subdir3'), 'file1_in_subdir3 missing from new collection'
    assert page.has_text?('file2_in_subdir3'), 'file2_in_subdir3 missing from new collection'
    assert page.has_no_text?('file1_in_subdir4'), 'file1_in_subdir4 found in new collection'
    assert page.has_no_text?('file2_in_subdir4'), 'file2_in_subdir4 found in new collection'

    # Make sure we're not still on the old collection page.
    refute_match(%r{/collections/#{col['uuid']}}, page.current_url)
  end

  test "remove a file from collection using checkbox and dropdown option" do
    need_selenium 'to confirm unlock'

    visit page_with_token('active', '/collections/zzzzz-4zz18-a21ux3541sxa8sf')
    assert(page.has_text?('file1'), 'file not found - file1')

    unlock_collection

    # remove first file
    input_files = page.all('input[type=checkbox]')
    input_files[0].click

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Remove selected files'
    end

    assert(page.has_no_text?('file1'), 'file found - file')
    assert(page.has_text?('file2'), 'file not found - file2')
  end

  test "remove a file in collection using trash icon" do
    need_selenium 'to confirm unlock'

    visit page_with_token('active', '/collections/zzzzz-4zz18-a21ux3541sxa8sf')
    assert(page.has_text?('file1'), 'file not found - file1')

    unlock_collection

    first('.fa-trash-o').click
    accept_alert

    assert(page.has_no_text?('file1'), 'file found - file')
    assert(page.has_text?('file2'), 'file not found - file2')
  end

  test "rename a file in collection" do
    need_selenium 'to confirm unlock'

    visit page_with_token('active', '/collections/zzzzz-4zz18-a21ux3541sxa8sf')

    unlock_collection

    within('.collection_files') do
      first('.fa-pencil').click
      find('.editable-input input').set('file1renamed')
      find('.editable-submit').click
    end

    assert(page.has_text?('file1renamed'), 'file not found - file1renamed')
  end

  test "remove/rename file options not presented if user cannot update a collection" do
    # visit a publicly accessible collection as 'spectator'
    visit page_with_token('spectator', '/collections/zzzzz-4zz18-uukreo9rbgwsujr')

    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li', text: 'Create new collection with selected files'
      assert_no_selector 'li', text: 'Remove selected files'
    end

    within('.collection_files') do
      assert(page.has_text?('GNU_General_Public_License'), 'file not found - GNU_General_Public_License')
      assert_nil first('.fa-pencil')
      assert_nil first('.fa-trash-o')
    end
  end

  test "unlock collection to modify files" do
    need_selenium 'to confirm remove'

    collection = api_fixture('collections')['collection_owned_by_active']

    # On load, collection is locked, and upload tab, rename and remove options are disabled
    visit page_with_token('active', "/collections/#{collection['uuid']}")

    assert_selector 'a[data-toggle="disabled"]', text: 'Upload'

    within('.collection_files') do
      file_ctrls = page.all('.btn-collection-file-control')
      assert_equal 2, file_ctrls.size
      assert_equal true, file_ctrls[0]['class'].include?('disabled')
      assert_equal true, file_ctrls[1]['class'].include?('disabled')
      find('input[type=checkbox]').click
    end

    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li.disabled', text: 'Remove selected files'
      assert_selector 'li', text: 'Create new collection with selected files'
    end

    unlock_collection

    assert_no_selector 'a[data-toggle="disabled"]', text: 'Upload'
    assert_selector 'a', text: 'Upload'

    within('.collection_files') do
      file_ctrls = page.all('.btn-collection-file-control')
      assert_equal 2, file_ctrls.size
      assert_equal false, file_ctrls[0]['class'].include?('disabled')
      assert_equal false, file_ctrls[1]['class'].include?('disabled')

      # previous checkbox selection won't result in firing a new event;
      # undo and redo checkbox to fire the selection event again
      find('input[type=checkbox]').click
      find('input[type=checkbox]').click
    end

    click_button 'Selection'
    within('.selection-action-container') do
      assert_no_selector 'li.disabled', text: 'Remove selected files'
      assert_selector 'li', text: 'Remove selected files'
    end
  end

  def unlock_collection
    first('.lock-collection-btn').click
    accept_alert
  end
end
