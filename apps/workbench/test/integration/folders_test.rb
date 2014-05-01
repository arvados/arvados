require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class FoldersTest < ActionDispatch::IntegrationTest

  test 'Find a folder and edit its description' do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token 'active', '/'
    find('nav a', text: 'Folders').click
    find('tr', text: 'A Folder').
      find('a,button', text: 'Show').
      click
    within('.panel', text: api_fixture('groups')['afolder']['name']) do
      find('span', text: api_fixture('groups')['afolder']['name']).click
      find('.glyphicon-ok').click
      find('.btn', text: 'Edit description').click
      find('.editable-input textarea').set('I just edited this.')
      find('.editable-submit').click
    end
    #find('.panel', text: 'I just edited this.')
  end

end
