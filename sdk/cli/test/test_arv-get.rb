require 'minitest/autorun'
require 'json'
require 'yaml'

# Black box tests for 'arv get' command.
class TestArvGet < Minitest::Test
  # UUID for an Arvados object that does not exist
  NON_EXISTANT_OBJECT_UUID = "qr1hi-tpzed-p8yk1lihjsgwew0"
  # Name of field of Arvados object that can store any (textual) value
  STORED_VALUE_FIELD_NAME = "name"
  # Name of UUID field of Arvados object
  UUID_FIELD_NAME = "uuid"
  # Name of an invalid field of Arvados object
  INVALID_FIELD_NAME = "invalid"

  # Tests that a valid Arvados object can be retrieved in JSON format using:
  # `arv get [uuid] --format json`.
  def test_get_valid_object_json_format()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid, '--format', 'json')
    end
    assert_empty(err)
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
  end

  # Tests that a valid Arvados object can be retrieved in YAML format using:
  # `arv get [uuid] --format yaml`.
  def test_get_valid_object_yaml_format()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid, '--format', 'yaml')
    end
    assert_empty(err)
    arv_object = parse_yaml_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
  end

  # Tests that a valid Arvados object can be retrieved in a supported format
  # using: `arv get [uuid]`. Given all other `arv foo` commands return JSON
  # when no format is specified, JSON should be expected in this case.
  def test_get_valid_object_no_format()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid)
    end
    assert_empty(err)
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
  end

  # Tests that a subset of all fields of a valid Arvados object can be retrieved
  # using: `arv get [uuid] [fields...]`.
  def test_get_valid_object_with_specific_valid_fields()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid, STORED_VALUE_FIELD_NAME, UUID_FIELD_NAME, "--format", "json")
    end
    assert_empty(err)
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
    assert(has_field_with_value(arv_object, UUID_FIELD_NAME, uuid))
  end

  # Tests that the valid field is retrieved when both a valid and invalid field
  # are requested from a valid Arvados object, using:
  # `arv get [uuid] [fields...]`.
  def test_get_valid_object_with_both_specific_valid_and_invalid_fields()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid, STORED_VALUE_FIELD_NAME, INVALID_FIELD_NAME, "--format", "json")
    end
    assert_empty(err)
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
    refute(has_field_with_value(arv_object, INVALID_FIELD_NAME, stored_value))
  end

  # Tests that no fields are retreived when no valid fields are requested from
  # a valid Arvados object, using: `arv get [uuid] [fields...]`.
  def test_get_valid_object_with_no_specific_valid_fields()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid, INVALID_FIELD_NAME, "--format", "json")
    end
    assert_empty(err)
    arv_object = parse_json_arv_object(out)
    assert_equal(0, arv_object.fixnum)
  end

  # Tests that an valid Arvados object is not retrieved when specifying an
  # invalid format: `arv get [uuid] --format invalid`.
  def test_get_valid_object_invalid_format()
    stored_value = __method__
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      arv_get(uuid, '--format', 'invalid')
    end
    refute_empty(err)
    assert_empty(out)
  end

  # Tests that an invalid (non-existant) Arvados object is not retrieved using:
  # using: `arv get [non-existant-uuid]`.
  def test_get_invalid_object()
    out, err = capture_subprocess_io do
      arv_get(NON_EXISTANT_OBJECT_UUID, "--format", "json")
    end
    refute_empty(err)
    assert_empty(out)
  end

  protected
  # Runs 'arv get <varargs>' with given arguments.
  def arv_get(*args)
    system(['./bin/arv', 'arv get'], *args)
  end

  # Creates an Arvados object that stores a given value. Returns the uuid of the
  # created object.
  def create_arv_object_with_value(value)
      out, err = capture_subprocess_io do
        # Write (without redirect)
        system(['./bin/arv', "arv tag add #{value} --object testing"])
      end
      if err.length > 0
        raise "Could not create Arvados object with given value"
      end
      return out
  end

  # Parses the given JSON representation of an Arvados object, returning
  # an equivalent Ruby representation (a hash map).
  def parse_json_arv_object(arvObjectAsJson)
    begin
      parsed = JSON.parse(arvObjectAsJson)
      assert(parsed.instance_of?(Hash))
      return parsed
    rescue JSON::ParserError => e
      raise "Invalid JSON representation of Arvados object"
    end
  end

  # Parses the given JSON representation of an Arvados object, returning
  # an equivalent Ruby representation (a hash map).
  def parse_yaml_arv_object(arvObjectAsYaml)
    begin
      parsed = YAML.load(arvObjectAsYaml)
      assert(parsed.instance_of?(Hash))
      return parsed
    rescue
      raise "Invalid YAML representation of Arvados object"
    end
  end

  # Checks whether the given Arvados object has the given expected value for the
  # specified field.
  def has_field_with_value(arvObjectAsHash, fieldName, expectedValue)
    if !arvObjectAsHash.has_key?(fieldName)
      return false
    end
    return (arvObjectAsHash[fieldName] == expectedValue)
  end
end
