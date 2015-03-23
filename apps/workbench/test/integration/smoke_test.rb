require 'integration_helper'
require 'uri'

class SmokeTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  def assert_visit_success(allowed=[200])
    assert_includes(allowed, status_code,
                    "#{current_url} returned #{status_code}, not one of " +
                    allowed.inspect)
  end

  def all_links_in(find_spec, text_regexp=//)
    all(find_spec + ' a').collect { |tag|
      if tag[:href].nil? or tag[:href].empty? or (tag.text !~ text_regexp)
        nil
      elsif tag[:'data-remote']
        # these don't necessarily work with format=html
        nil
      else
        url = URI(tag[:href])
        url.host.nil? ? url.path : nil
      end
    }.compact
  end

  test "all first-level links succeed" do
    visit page_with_token('active_trustedclient', '/')
    assert_visit_success
    click_link 'notifications-menu'
    urls = [all_links_in('nav'),
            all_links_in('.navbar', /^Manage /)].flatten
    seen_urls = ['/']
    while not (url = urls.shift).nil?
      next if seen_urls.include? url
      visit url
      seen_urls << url
      assert_visit_success
      # Uncommenting the line below lets you crawl the entire site for a
      # more thorough test.
      # urls += all_links_in('body')
    end
  end
end
