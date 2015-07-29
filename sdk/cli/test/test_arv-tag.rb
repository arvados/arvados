require 'minitest/autorun'
require 'digest/md5'
require 'json'

def assert_failure *args
  assert_equal false, *args
end

class TestArvTag < Minitest::Test

  def test_no_args
    skip "Waiting until #4534 is implemented"

    # arv-tag exits with failure if run with no args
    out, err = capture_subprocess_io do
      assert_equal false, arv_tag
    end
    assert_empty out
    assert_match /^usage:/i, err
  end

  # Test adding and removing a single tag on a single object.
  def test_single_tag_single_obj
    skip "TBD"

    # Add a single tag.
    tag_uuid, err = capture_subprocess_io do
      assert arv_tag '--short', 'add', 'test_tag1', '--object', 'uuid1'
    end
    assert_empty err

    out, err = capture_subprocess_io do
      assert arv 'link', 'show', '--uuid', tag_uuid.rstrip
    end

    assert_empty err
    link = JSON.parse out
    assert_tag link, 'test_tag1', 'uuid1'

    # Remove the tag.
    out, err = capture_subprocess_io do
      assert arv_tag 'remove', 'test_tag1', '--object', 'uuid1'
    end

    assert_empty err
    links = JSON.parse out
    assert_equal 1, links.length
    assert_tag links[0], 'test_tag1', 'uuid1'

    # Verify that the link no longer exists.
    out, err = capture_subprocess_io do
      assert_equal false, arv('link', 'show', '--uuid', links[0]['uuid'])
    end

    assert_equal "Error: Path not found\n", err
  end

  # Test adding and removing a single tag with multiple objects.
  def test_single_tag_multi_objects
    skip "TBD"

    out, err = capture_subprocess_io do
      assert arv_tag('add', 'test_tag1',
                     '--object', 'uuid1',
                     '--object', 'uuid2',
                     '--object', 'uuid3')
    end
    assert_empty err

    out, err = capture_subprocess_io do
      assert arv 'link', 'list', '--where', '{"link_class":"tag","name":"test_tag1"}'
    end

    assert_empty err
    json_out = JSON.parse out
    links = json_out['items'].sort { |a,b| a['head_uuid'] <=> b['head_uuid'] }
    assert_equal 3, links.length
    assert_tag links[0], 'test_tag1', 'uuid1'
    assert_tag links[1], 'test_tag1', 'uuid2'
    assert_tag links[2], 'test_tag1', 'uuid3'

    out, err = capture_subprocess_io do
      assert arv_tag('remove', 'test_tag1',
                     '--object', 'uuid1',
                     '--object', 'uuid2',
                     '--object', 'uuid3')
    end
    assert_empty err

    out, err = capture_subprocess_io do
      assert arv 'link', 'list', '--where', '{"link_class":"tag","name":"test_tag1"}'
    end

    assert_empty err
    assert_empty out
  end

  protected
  def arv_tag(*args)
    system ['./bin/arv-tag', 'arv-tag'], *args
  end

  def arv(*args)
    system ['./bin/arv', 'arv'], *args
  end

  def assert_tag(link, name, head_uuid)
    assert_equal 'tag',     link['link_class']
    assert_equal name,      link['name']
    assert_equal head_uuid, link['head_uuid']
  end
end
