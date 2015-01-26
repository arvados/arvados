# http://guides.rubyonrails.org/v3.2.13/performance_testing.html

require 'test_helper'
require 'rails/performance_test_help'
require 'performance_test_helper'
require 'selenium-webdriver'
require 'headless'

class BrowsingTest < WorkbenchPerformanceTest
  self.profile_options = { :runs => 5,
                           :metrics => [:wall_time],
                           :output => 'tmp/performance',
                           :formats => [:flat] }

  setup do
    need_javascript
  end

  test "home page" do
    visit_page_with_token
    assert_text 'Dashboard'
    assert_selector 'a', text: 'Run a pipeline'
  end

  test "search for hash" do
    visit_page_with_token
    assert_text 'Dashboard'

    assert_selector '.navbar-fixed-top'
    assert_triggers_dom_event 'shown.bs.modal' do
      within '.navbar-fixed-top' do
        find_field('search').set 'hash'
        find('.glyphicon-search').click
      end
    end

    # In the search dialog now. Expect at least one item in the result display.
    within '.modal-content' do
      assert_text 'All projects'
      assert_text 'Search'
      assert_selector '.selectable[data-object-uuid]'
      click_button 'Cancel'
    end
  end
end
