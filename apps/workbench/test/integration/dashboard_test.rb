require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class DashboardTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  # verify active and finished pipelines, and collections in the dashboard
  def verify_dashboard user, num_finished, num_running, num_collections, exact_counts
    verify_active_pipelines_panel num_running, exact_counts
    verify_finished_pipelines_panel num_finished, exact_counts
    verify_recent_collections_panel num_collections, exact_counts
  end

  # verify recently finished pipelines panel in dashboard
  def verify_finished_pipelines_panel num_finished, exact_counts
    assert(page.has_text?('Recently finished pipelines'), 'Recently finished pipelines - not found on dashboard')

    # The recently finished pipelines panel pulls in 8 pipelines at a time
    num_pages = num_finished/8 + 1
    (0..num_pages).each do |i|
      within('.arv-recently-finished-pipelines') do
        page.execute_script "window.scrollBy(0,999000)"
        begin
          wait_for_ajax
        rescue
        end
      end
    end

    expected_names = []
    (0..num_finished-1).each do |i|
      expected_names << "pipeline_#{i.to_s}"
    end

    found_names = []
    found_obj_count = 0
    within('.arv-recently-finished-pipelines') do
      objs_found = page.all('tr[data-kind="arvados#pipelineInstance"]')
      found_obj_count = objs_found.count

      (0..found_obj_count-1).each do |i|
        name = objs_found[i].text.split[0]
        found_names << name
      end
    end

    assert_equal true, (found_names - expected_names).empty?,
                 "did not find all names: #{found_names - expected_names}"

    if exact_counts
      assert_equal num_finished, found_obj_count,
          "Expected #{num_finished} but found #{found_obj_count} finished pipelines"
    else
      assert_equal true, (found_obj_count >= num_finished),
          "Expected at least #{num_finished} but found #{found_obj_count} finished pipelines"
    end
  end

  # verify active pipelines panel in dashboard
  def verify_active_pipelines_panel num_running, exact_counts
    assert(page.has_text?('Active pipelines'), 'Active pipelines - not found on dashboard')

    found_obj_count = 0
    within('.arv-dashboard-running-pipelines') do
      objs_found = page.all('div[class="dashboard-panel-info-row"]')
      found_obj_count = objs_found.count
    end

    if exact_counts
      assert_equal num_running, found_obj_count,
          "Expected #{num_running} but found #{found_obj_count} running pipelines"
    else
      assert_equal true, (found_obj_count >= num_running),
          "Expected at least #{num_running} but found #{found_obj_count} running pipelines"
    end
  end

  # verify recent collections panel in dashboard
  def verify_recent_collections_panel num_collections, exact_counts
    assert(page.has_text?('Recent collections'), 'Recent collections - not found on dashboard')

    found_obj_count = 0
    within('.arv-dashboard-recent-collections') do
      objs_found = page.all('div[class="dashboard-panel-info-row"]')
      found_obj_count = objs_found.count
    end

    if exact_counts
      assert_equal num_collections, found_obj_count,
          "Expected #{num_collections} but found #{found_obj_count} recent collections"
    else
      assert_equal true, (found_obj_count >= num_collections),
          "Expected at least #{num_collections} but found #{found_obj_count} recent collections"
    end
  end

  [
    ['user_with_no_objs', 0, 0, 0, true],
    ['user_with_5_finished_pipelines', 5, 0, 0, true],
    ['user_with_1_running_25_finished_pipelines', 25, 1, 0, true],
    ['active', 0, 1, 6, false],
    ['admin', 35, 2, 8, false],
  ].each do |user, num_finished, num_running, num_collections, exact_counts|
    test "dashboard recently finished pipelines panel for #{user} with #{num_finished} finished,
          #{num_running} running, #{num_collections} collections, #{exact_counts}" do
      visit page_with_token(user)
      wait_for_ajax
      verify_dashboard user, num_finished, num_running, num_collections, exact_counts
    end
  end
end
