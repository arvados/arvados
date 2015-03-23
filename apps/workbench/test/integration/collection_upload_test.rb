require 'integration_helper'

class CollectionUploadTest < ActionDispatch::IntegrationTest
  setup do
    testfiles.each do |filename, content|
      open(testfile_path(filename), 'w') do |io|
        io.write content
      end
    end
  end

  teardown do
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

  test "No Upload tab on non-writable collection" do
    need_javascript
    visit(page_with_token 'active',
          '/collections/'+api_fixture('collections')['user_agreement']['uuid'])
    assert_no_selector '.nav-tabs Upload'
  end

  test "Upload two empty files with the same name" do
    need_selenium "to make file uploads work"
    visit page_with_token 'active', sandbox_path
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

  test "Upload non-empty files, report errors" do
    need_selenium "to make file uploads work"
    visit page_with_token 'active', sandbox_path
    find('.nav-tabs a', text: 'Upload').click
    attach_file 'file_selector', testfile_path('a')
    attach_file 'file_selector', testfile_path('foo.txt')
    assert_selector 'button:not([disabled])', text: 'Start'
    click_button 'Start'
    if "test environment does not have a keepproxy yet, see #4534" != "fixed"
      using_wait_time 20 do
        assert_text :visible, 'error'
      end
    else
      assert_text :visible, 'Done!'
      visit sandbox_path+'.json'
      assert_match /_text":"\. 0cc1\S+ 0:1:a\\n\. acbd\S+ 0:3:foo.txt\\n"/, body
    end
  end

  test "Report mixed-content error" do
    skip 'Test suite does not use TLS'
    need_selenium "to make file uploads work"
    begin
      use_token :admin
      proxy = KeepService.find(api_fixture('keep_services')['proxy']['uuid'])
      proxy.update_attributes service_ssl_flag: false
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
    begin
      use_token :admin
      proxy = KeepService.find(api_fixture('keep_services')['proxy']['uuid'])
      # Even if you somehow do port>2^16, surely nx.example.net won't respond
      proxy.update_attributes service_host: 'nx.example.net', service_port: 99999
    end
    visit page_with_token 'active', sandbox_path
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
end
