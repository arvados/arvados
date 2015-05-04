require 'integration_helper'
require 'helpers/repository_stub_helper'
require 'helpers/share_object_helper'

class RepositoriesTest < ActionDispatch::IntegrationTest
  include RepositoryStubHelper
  include ShareObjectHelper

  reset_api_fixtures :after_each_test, false

  setup do
    need_javascript
  end

  test "browse repository from jobs#show" do
    sha1 = api_fixture('jobs')['running']['script_version']
    _, fakecommit, fakefile =
      stub_repo_content sha1: sha1, filename: 'crunch_scripts/hash'
    show_object_using 'active', 'jobs', 'running', sha1
    click_on api_fixture('jobs')['running']['script']
    assert_text fakefile
    click_on 'crunch_scripts'
    assert_selector 'td a', text: 'hash'
    click_on 'foo'
    assert_selector 'td a', text: 'crunch_scripts'
    click_on sha1
    assert_text fakecommit

    show_object_using 'active', 'jobs', 'running', sha1
    click_on 'active/foo'
    assert_selector 'td a', text: 'crunch_scripts'

    show_object_using 'active', 'jobs', 'running', sha1
    click_on sha1
    assert_text fakecommit
  end

  test "browse using arv-git-http" do
    repo = api_fixture('repositories')['foo']
    portfile =
      File.expand_path('../../../../../tmp/arv-git-httpd-ssl.port', __FILE__)
    gitsslport = File.read(portfile)
    Repository.any_instance.
      stubs(:http_fetch_url).
      returns "https://localhost:#{gitsslport}/#{repo['name']}.git"
    commit_sha1 = '1de84a854e2b440dc53bf42f8548afa4c17da332'
    visit page_with_token('active', "/repositories/#{repo['uuid']}/commit/#{commit_sha1}")
    assert_text "Date:   Tue Mar 18 15:55:28 2014 -0400"
    visit page_with_token('active', "/repositories/#{repo['uuid']}/tree/#{commit_sha1}")
    assert_selector "tbody td a", "foo"
    assert_text "12 bytes"
  end
end
