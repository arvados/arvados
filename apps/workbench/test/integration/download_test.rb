require 'integration_helper'
require 'helpers/download_helper'

class DownloadTest < ActionDispatch::IntegrationTest
  include KeepWebConfig

  setup do
    use_keep_web_config

    # Make sure Capybara can download files.
    need_selenium 'for downloading', :selenium_with_download
    DownloadHelper.clear

    # Keep data isn't populated by fixtures, so we have to write any
    # data we expect to read.
    ['foo', 'w a z', "Hello world\n"].each do |data|
      md5 = `echo -n #{data.shellescape} | arv-put --no-progress --raw -`
      assert_match /^#{Digest::MD5.hexdigest(data)}/, md5
      assert $?.success?, $?
    end
  end

  ['uuid', 'portable_data_hash'].each do |id_type|
    test "preview from keep-web by #{id_type} using a reader token" do
      uuid_or_pdh = api_fixture('collections')['foo_file'][id_type]
      token = api_fixture('api_client_authorizations')['active_all_collections']['api_token']
      visit "/collections/download/#{uuid_or_pdh}/#{token}/"
      within "#collection_files" do
        click_link 'foo'
      end
      assert_no_selector 'a'
      assert_text 'foo'
    end

    test "preview anonymous content from keep-web by #{id_type}" do
      Rails.configuration.anonymous_user_token =
        api_fixture('api_client_authorizations')['anonymous']['api_token']
      uuid_or_pdh =
        api_fixture('collections')['public_text_file'][id_type]
      visit "/collections/#{uuid_or_pdh}"
      within "#collection_files" do
        find('[title~=View]').click
      end
      assert_no_selector 'a'
      assert_text 'Hello world'
    end

    test "download anonymous content from keep-web by #{id_type}" do
      Rails.configuration.anonymous_user_token =
        api_fixture('api_client_authorizations')['anonymous']['api_token']
      uuid_or_pdh =
        api_fixture('collections')['public_text_file'][id_type]
      visit "/collections/#{uuid_or_pdh}"
      within "#collection_files" do
        find('[title~=Download]').click
      end
      wait_for_download 'Hello world.txt', "Hello world\n"
    end
  end

  test "download from keep-web using a session token" do
    uuid = api_fixture('collections')['w_a_z_file']['uuid']
    token = api_fixture('api_client_authorizations')['active']['api_token']
    visit page_with_token('active', "/collections/#{uuid}")
    within "#collection_files" do
      find('[title~=Download]').click
    end
    wait_for_download 'w a z', 'w a z'
  end

  def wait_for_download filename, expect_data
    data = nil
    tries = 0
    while tries < 20
      sleep 0.1
      tries += 1
      data = File.read(DownloadHelper.path.join filename) rescue nil
    end
    assert_equal expect_data, data
  end

  # TODO(TC): test "view pages hosted by keep-web, using session
  # token". We might persuade selenium to send
  # "collection-uuid.dl.example" requests to localhost by configuring
  # our test nginx server to work as its forward proxy. Until then,
  # we're relying on the "Redirect to keep_web_url via #{id_type}"
  # test in CollectionsControllerTest (and keep-web's tests).
end
