require 'minitest/autorun'
require 'digest/md5'
require 'json'

def assert_failure *args
  assert_equal false, *args
end

class TestArvTag < Minitest::Test

  def test_no_args
    # arv-tag exits with failure if run with no args
    out, err = capture_subprocess_io do
      assert_equal false, arv_tag
    end
    assert_empty out
    assert_match /^usage:/i, err
  end

  # Test adding and removing a single tag on a single object.
  def test_single_tag_single_obj
    tag_uuid, err = capture_subprocess_io do
      assert arv_tag 'add', 'test_tag1', '--object', 'uuid1'
    end
    assert_empty err

    out, err = capture_subprocess_io do
      assert arv '-h', 'link', 'show', '--uuid', tag_uuid.rstrip
    end

    assert_empty err
    tag = JSON.parse out
    assert_equal 'test_tag1', tag['name']
    assert_equal 'tag',       tag['link_class']
    assert_equal 'uuid1',     tag['head_uuid']

    out, err = capture_subprocess_io do
      assert arv_tag '-h', 'remove', 'test_tag1', '--object', 'uuid1'
    end

    assert_empty err
    tag = JSON.parse out
    assert_equal 'test_tag1', tag[0]['name']
    assert_equal 'tag',       tag[0]['link_class']
    assert_equal 'uuid1',     tag[0]['head_uuid']

    out, err = capture_subprocess_io do
      assert_equal false, arv('-h', 'link', 'show', '--uuid', tag[0]['uuid'])
    end

    assert_equal "Error: Path not found\n", err
  end

  # Test adding and removing a single tag with multiple objects.
  def test_single_tag_multi_objects
    out, err = capture_subprocess_io do
      assert arv_tag('add', 'test_tag1',
                     '--object', 'uuid1',
                     '--object', 'uuid2',
                     '--object', 'uuid3')
    end
    assert_empty err

    out, err = capture_subprocess_io do
      assert arv '-h', 'link', 'list', '--where', '{"link_class":"tag","name":"test_tag1"}'
    end

    assert_empty err
    json_out = JSON.parse out
    links = json_out['items'].sort { |a,b| a['head_uuid'] <=> b['head_uuid'] }
    assert_equal 'test_tag1', links[0]['name']
    assert_equal 'tag',       links[0]['link_class']
    assert_equal 'uuid1',     links[0]['head_uuid']
    assert_equal 'test_tag1', links[1]['name']
    assert_equal 'tag',       links[1]['link_class']
    assert_equal 'uuid2',     links[1]['head_uuid']
    assert_equal 'test_tag1', links[2]['name']
    assert_equal 'tag',       links[2]['link_class']
    assert_equal 'uuid3',     links[2]['head_uuid']

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

  # Test adding and removing multiple tags with multiple objects.
  def test_multi_tag_multi_objects
    out, err = capture_subprocess_io do
      assert arv_tag('add', 'test_tag1', 'test_tag2', 'test_tag3',
                     '--object', 'uuid1',
                     '--object', 'uuid2',
                     '--object', 'uuid3')
    end

    out, err = capture_subprocess_io do
      assert arv '-h', 'link', 'list', '--where', '{"link_class":"tag"}'
    end

    assert_empty err
    json_out = JSON.parse out
    links = json_out['items'].sort { |a,b|
      a['name'] <=> b['name'] or
      a['head_uuid'] <=> b['head_uuid']
    }

    assert_equal 'test_tag1', links[0]['name']
    assert_equal 'tag',       links[0]['link_class']
    assert_equal 'uuid1',     links[0]['head_uuid']
    assert_equal 'test_tag1', links[1]['name']
    assert_equal 'tag',       links[1]['link_class']
    assert_equal 'uuid2',     links[1]['head_uuid']
    assert_equal 'test_tag1', links[2]['name']
    assert_equal 'tag',       links[2]['link_class']
    assert_equal 'uuid3',     links[2]['head_uuid']

    assert_equal 'test_tag2', links[3]['name']
    assert_equal 'tag',       links[3]['link_class']
    assert_equal 'uuid1',     links[3]['head_uuid']
    assert_equal 'test_tag2', links[4]['name']
    assert_equal 'tag',       links[4]['link_class']
    assert_equal 'uuid2',     links[4]['head_uuid']
    assert_equal 'test_tag2', links[5]['name']
    assert_equal 'tag',       links[5]['link_class']
    assert_equal 'uuid3',     links[5]['head_uuid']

    assert_equal 'test_tag3', links[6]['name']
    assert_equal 'tag',       links[6]['link_class']
    assert_equal 'uuid1',     links[6]['head_uuid']
    assert_equal 'test_tag3', links[7]['name']
    assert_equal 'tag',       links[7]['link_class']
    assert_equal 'uuid2',     links[7]['head_uuid']
    assert_equal 'test_tag3', links[8]['name']
    assert_equal 'tag',       links[8]['link_class']
    assert_equal 'uuid3',     links[8]['head_uuid']

  end

  protected
  def arv_tag(*args)
    system ['./bin/arv-tag', 'arv-tag'], *args
  end

  def arv(*args)
    system ['./bin/arv', 'arv'], *args
  end
end
