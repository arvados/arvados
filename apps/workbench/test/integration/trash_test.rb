# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'

class TrashTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  test "trash page" do
    deleted = api_fixture('collections')['deleted_on_next_sweep']
    expired1 = api_fixture('collections')['unique_expired_collection']
    expired2 = api_fixture('collections')['unique_expired_collection2']

    # visit trash page
    visit page_with_token('active', "/trash")

    assert_text deleted['name']
    assert_text deleted['uuid']
    assert_text deleted['portable_data_hash']
    assert_text expired1['name']
    assert_no_text expired2['name']   # not readable by this user
    assert_no_text 'foo_file'         # not trash

    # Un-trash one item using selection dropdown
    within('tr', text: deleted['name']) do
      first('input').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Un-trash selected items'
    end

    wait_for_ajax

    assert_text expired1['name']      # this should still be there
    assert_no_text deleted['name']    # this should no longer be here

    # Un-trash another item using the recycle button
    within('tr', text: expired1['name']) do
      first('.fa-recycle').click
    end

    wait_for_ajax

    assert_text "The collection with UUID #{expired1['uuid']} is in the trash"

    click_on "Click here to untrash '#{expired1['name']}'"

    # verify that the two un-trashed items are now shown in /collections page
    visit page_with_token('active', "/collections")
    assert_text deleted['uuid']
    assert_text expired1['uuid']
    assert_no_text expired2['uuid']
  end

  ["button","selection"].each do |method|
    test "trashed projects using #{method}" do
      deleted = api_fixture('groups')['trashed_project']
      aproject = api_fixture('groups')['aproject']

      # verify that the un-trashed item are missing in /groups page
      visit page_with_token('active', "/projects/zzzzz-tpzed-xurymjxw79nv3jz")
      click_on "Subprojects"
      assert_no_text deleted['name']

      # visit trash page
      visit page_with_token('active', "/trash")
      click_on "Trashed projects"

      assert_text deleted['name']
      assert_text deleted['uuid']
      assert_no_text aproject['name']
      assert_no_text aproject['uuid']

      # Un-trash item
      if method == "button"
        within('tr', text: deleted['uuid']) do
          first('.fa-recycle').click
        end
        assert_text "The group with UUID #{deleted['uuid']} is in the trash"
        click_on "Click here to untrash '#{deleted['name']}'"
      else
        within('tr', text: deleted['uuid']) do
          first('input').click
        end
        click_button 'Selection...'
        within('.selection-action-container') do
          click_link 'Un-trash selected items'
        end
        wait_for_ajax
        assert_no_text deleted['uuid']
      end

      # check that the un-trashed item are now shown on parent project page
      visit page_with_token('active', "/projects/zzzzz-tpzed-xurymjxw79nv3jz")
      click_on "Subprojects"
      assert_text deleted['name']
      assert_text aproject['name']

      # Trash another item
      if method == "button"
        within('tr', text: aproject['name']) do
          first('.fa-trash-o').click
        end
      else
        within('tr', text: aproject['name']) do
          first('input').click
        end
        click_button 'Selection'
        within('.selection-action-container') do
          click_link 'Remove selected'
        end
      end

      wait_for_ajax
      assert_no_text aproject['name']
      visit current_path
      assert_no_text aproject['name']

      # visit trash page
      visit page_with_token('active', "/trash")
      click_on "Trashed projects"

      assert_text aproject['name']
      assert_text aproject['uuid']
    end
  end

  test "trash page with search" do
    deleted = api_fixture('collections')['deleted_on_next_sweep']
    expired = api_fixture('collections')['unique_expired_collection']

    visit page_with_token('active', "/trash")

    assert_text deleted['name']
    assert_text deleted['uuid']
    assert_text deleted['portable_data_hash']
    assert_text expired['name']

    page.find_field('Search trash').set 'expired'

    assert_no_text deleted['name']
    assert_text expired['name']

    page.find_field('Search trash').set deleted['portable_data_hash'][0..9]

    assert_no_text expired['name']
    assert_text deleted['name']
    assert_text deleted['uuid']
    assert_text deleted['portable_data_hash']

    click_button 'Selection...'
    within('.selection-action-container') do
      assert_selector 'li.disabled', text: 'Un-trash selected items'
    end

    first('input').click

    click_button 'Selection...'
    within('.selection-action-container') do
      assert_selector 'li', text: 'Un-trash selected items'
      assert_selector 'li.disabled', text: 'Un-trash selected items'
    end
  end
end
