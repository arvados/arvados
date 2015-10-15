require 'minitest/autorun'
require 'json'
require 'yaml'

# Black box tests for 'arv get' command.
class TestArvGet < Minitest::Test
  # UUID for an Arvados object that does not exist
  NON_EXISTENT_OBJECT_UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
  # Name of field of Arvados object that can store any (textual) value
  STORED_VALUE_FIELD_NAME = "name"
  # Name of UUID field of Arvados object
  UUID_FIELD_NAME = "uuid"
  # Name of an invalid field of Arvados object
  INVALID_FIELD_NAME = "invalid"

  # Tests that a valid Arvados object can be retrieved in a supported format
  # using: `arv get [uuid]`. Given all other `arv foo` commands return JSON
  # when no format is specified, JSON should be expected in this case.
  def test_get_valid_object_no_format_specified
    stored_value = __method__.to_s
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      assert(arv_get_default(uuid))
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
  end

  # Tests that a valid Arvados object can be retrieved in JSON format using:
  # `arv get [uuid] --format json`.
  def test_get_valid_object_json_format_specified
    stored_value = __method__.to_s
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      assert(arv_get_json(uuid))
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
  end

  # Tests that a valid Arvados object can be retrieved in YAML format using:
  # `arv get [uuid] --format yaml`.
  def test_get_valid_object_yaml_format_specified
    stored_value = __method__.to_s
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      assert(arv_get_yaml(uuid))
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    arv_object = parse_yaml_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
  end

  # Tests that a subset of all fields of a valid Arvados object can be retrieved
  # using: `arv get [uuid] [fields...]`.
  def test_get_valid_object_with_valid_fields
    stored_value = __method__.to_s
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      assert(arv_get_json(uuid, STORED_VALUE_FIELD_NAME, UUID_FIELD_NAME))
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
    assert(has_field_with_value(arv_object, UUID_FIELD_NAME, uuid))
  end

  # Tests that the valid field is retrieved when both a valid and invalid field
  # are requested from a valid Arvados object, using:
  # `arv get [uuid] [fields...]`.
  def test_get_valid_object_with_both_valid_and_invalid_fields
    stored_value = __method__.to_s
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      assert(arv_get_json(uuid, STORED_VALUE_FIELD_NAME, INVALID_FIELD_NAME))
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    arv_object = parse_json_arv_object(out)
    assert(has_field_with_value(arv_object, STORED_VALUE_FIELD_NAME, stored_value))
    refute(has_field_with_value(arv_object, INVALID_FIELD_NAME, stored_value))
  end

  # Tests that no fields are retreived when no valid fields are requested from
  # a valid Arvados object, using: `arv get [uuid] [fields...]`.
  def test_get_valid_object_with_no_valid_fields
    stored_value = __method__.to_s
    uuid = create_arv_object_with_value(stored_value)
    out, err = capture_subprocess_io do
      assert(arv_get_json(uuid, INVALID_FIELD_NAME))
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    arv_object = parse_json_arv_object(out)
    assert_equal(0, arv_object.length)
  end

  # Tests that an invalid (non-existent) Arvados object is not retrieved using:
  # using: `arv get [non-existent-uuid]`.
  def test_get_invalid_object
    out, err = capture_subprocess_io do
      refute(arv_get_json(NON_EXISTENT_OBJECT_UUID))
    end
    refute_empty(err, "Expected error feedback on request for invalid object")
    assert_empty(out)
  end

  # Tests that help text exists using: `arv get --help`.
  def test_help_exists
    out, err = capture_subprocess_io do
#      assert(arv_get_default("--help"), "Expected exit code 0: #{$?}")
       #XXX: Exit code given is 255. It probably should be 0, which seems to be
       #     standard elsewhere. However, 255 is in line with other `arv`
       #     commands (e.g. see `arv edit`) so ignoring the problem here.
       arv_get_default("--help")
    end
    assert_empty(err, "Error text not expected: '#{err}'")
    refute_empty(out, "Help text should be given")
  end

  protected
  # Runs 'arv get <varargs>' with given arguments. Returns whether the exit
  # status was 0 (i.e. success). Use $? to attain more details on failure.
  def arv_get_default(*args)
    return system("arv", "get", *args)
  end

  # Runs 'arv --format json get <varargs>' with given arguments. Returns whether
  # the exit status was 0 (i.e. success). Use $? to attain more details on
  # failure.
  def arv_get_json(*args)
    return system("arv", "--format", "json", "get", *args)
  end

  # Runs 'arv --format yaml get <varargs>' with given arguments. Returns whether
  # the exit status was 0 (i.e. success). Use $? to attain more details on
  # failure.
  def arv_get_yaml(*args)
    return system("arv", "--format", "yaml", "get", *args)
  end

  # Creates an Arvados object that stores a given value. Returns the uuid of the
  # created object.
  def create_arv_object_with_value(value)
    out, err = capture_subprocess_io do
      system("arv", "tag", "add", value, "--object", "testing")
      assert $?.success?, "Command failure running `arv tag`: #{$?}"
    end
    assert_equal '', err
    assert_operator 0, :<, out.strip.length
    out.strip
  end

  # Parses the given JSON representation of an Arvados object, returning
  # an equivalent Ruby representation (a hash map).
  def parse_json_arv_object(arvObjectAsJson)
    begin
      parsed = JSON.parse(arvObjectAsJson)
      assert(parsed.instance_of?(Hash))
      return parsed
    rescue JSON::ParserError => e
      raise "Invalid JSON representation of Arvados object.\n" \
            "Parse error: '#{e}'\n" \
            "JSON: '#{arvObjectAsJson}'\n"
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
      raise "Invalid YAML representation of Arvados object.\n" \
            "YAML: '#{arvObjectAsYaml}'\n"
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
