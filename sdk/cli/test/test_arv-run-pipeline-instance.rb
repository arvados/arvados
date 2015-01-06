require 'minitest/autorun'

class TestRunPipelineInstance < Minitest::Test
  def setup
  end

  def test_run_pipeline_instance_get_help
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      system ('arv-run-pipeline-instance -h')
    end
    assert_equal '', err
  end

  def test_run_pipeline_instance_with_no_such_option
    out, err = capture_subprocess_io do
      system ('arv-run-pipeline-instance --junk')
    end
    refute_equal '', err
  end

  def test_run_pipeline_instance_for_bogus_template_uuid
    out, err = capture_subprocess_io do
      # fails with error SSL_connect error because HOST_INSECURE is not being used
    	  # system ('arv-run-pipeline-instance --template bogus-abcde-fghijklmnopqrs input=c1bad4b39ca5a924e481008009d94e32+210')

      # fails with error: fatal: cannot load such file -- arvados
    	  # system ('./bin/arv-run-pipeline-instance --template bogus-abcde-fghijklmnopqrs input=c1bad4b39ca5a924e481008009d94e32+210')
    end
    #refute_equal '', err
    assert_equal '', err
  end

end
