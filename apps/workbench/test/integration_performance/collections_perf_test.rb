require 'integration_helper'

class CollectionsPerfTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = :rack_test

    skip "ENV variable RUN_INTG_PERF_TESTS with value 'y' is not found" if !ENV["RUN_INTG_PERF_TESTS"].andand.start_with? 'y'
  end

  def create_large_collection size, file_name_prefix
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e+0"

    i = 0
    until manifest_text.length > size do
      manifest_text << " 0:0:#{file_name_prefix}#{i.to_s}"
      i += 1
    end
    manifest_text << "\n"

    Collection.create! ({manifest_text: manifest_text})
  end

  [
    1000000,
    10000000,
    20000000,
  ].each do |size|
    test "Create and show large collection with manifest text of #{size}" do
      use_token :active
      new_collection = create_large_collection size, 'collection_file_name_with_prefix_'

      visit page_with_token('active', "/collections/#{new_collection.uuid}")

      assert_text new_collection.uuid
      assert(page.has_link?('collection_file_name_with_prefix_0'), "Collection page did not include file link")
    end
  end

  # This does not work with larger sizes because of need_javascript.
  # Just use one test with 100,000 for now.
  [
    100000,
  ].each do |size|
    test "Create, show, and update description for large collection with manifest text of #{size}" do
      need_javascript

      use_token :active
      new_collection = create_large_collection size, 'collection_file_name_with_prefix_'

      visit page_with_token('active', "/collections/#{new_collection.uuid}")

      assert_text new_collection.uuid
      assert(page.has_link?('collection_file_name_with_prefix_0'), "Collection page did not include file link")

      # edit description
      within('.arv-description-as-subtitle') do
        find('.fa-pencil').click
        find('.editable-input textarea').set('description for this large collection')
        find('.editable-submit').click
      end

      assert_text 'description for this large collection'
    end
  end

  [
    [1000000, 10000],
    [10000000, 10000],
    [20000000, 10000],
  ].each do |size1, size2|
    test "Create one large collection of #{size1} and one small collection of #{size2} and combine them" do
      use_token :active
      first_collection = create_large_collection size1, 'collection_file_name_with_prefix_1_'
      second_collection = create_large_collection size2, 'collection_file_name_with_prefix_2_'

      visit page_with_token('active', "/collections")

      assert_text first_collection.uuid
      assert_text second_collection.uuid

      within('tr', text: first_collection['uuid']) do
        find('input[type=checkbox]').click
      end

      within('tr', text: second_collection['uuid']) do
        find('input[type=checkbox]').click
      end

      click_button 'Selection...'
      within('.selection-action-container') do
        click_link 'Create new collection with selected collections'
      end

      assert(page.has_link?('collection_file_name_with_prefix_1_0'), "Collection page did not include file link")
    end
  end
end
