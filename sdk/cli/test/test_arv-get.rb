require 'minitest/autorun'

# Test for 'arv get' command.
class TestArvGet < Minitest::Test
  # Test setup.
  def setup
    # No setup required.
  end

  # Tests something... TODO.
  def test_get_TODO
    out, err = capture_subprocess_io do
      arv_get
    end

    assert_empty err
  end

  protected
  # Runs 'arv get <varargs>' with given arguments.
  def arv_get(*args)
    system ['./bin/arv', 'arv get'], *args
  end
end
