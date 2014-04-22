require 'test_helper'

class PipelineInstanceTest < ActiveSupport::TestCase

  test "check active and success for a pipeline in new state" do
    pi = pipeline_instances :new_pipeline

    assert !pi.active, 'expected active to be false for a new pipeline'
    assert !pi.success, 'expected success to be false for a new pipeline'
    assert !pi.state, 'expected state to be nil because the fixture had no state specified'

    # save the pipeline and expect state to be New
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::New, pi.state, 'expected state to be New for new pipeline'
    assert !pi.active, 'expected active to be false for a new pipeline'
    assert !pi.success, 'expected success to be false for a new pipeline'
  end

  test "update attributes for pipeline" do
    pi = pipeline_instances :new_pipeline

    # add a component with no input and expect state to be New
    component = {'script_parameters' => {"input_not_provided" => {"required" => true}}}
    pi.components['first'] = component
    components = pi.components
    pi.update_attribute 'components', pi.components
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::New, pi.state, 'expected state to be New after adding component with input'
    assert_equal pi.components.size, 1, 'expected one component'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false for a new pipeline'

    # add a component with no input not required
    component = {'script_parameters' => {"input_not_provided" => {"required" => false}}}
    pi.components['first'] = component
    components = pi.components
    pi.update_attribute 'components', pi.components
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Ready, pi.state, 'expected state to be Ready after adding component with input'
    assert_equal pi.components.size, 1, 'expected one component'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false for a new pipeline'

    # add a component with input and expect state to become Ready
    component = {'script_parameters' => {"input" => "yyyad4b39ca5a924e481008009d94e32+210"}}
    pi.components['first'] = component
    components = pi.components
    pi.update_attribute 'components', pi.components
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Ready, pi.state, 'expected state to be Ready after adding component with input'
    assert_equal pi.components.size, 1, 'expected one component'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false for a new pipeline'
   
    pi.active = true
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::RunningOnServer, pi.state, 'expected state to be RunningOnServer after updating active to true'
    assert pi.active, 'expected active to be true after update'
    assert !pi.success, 'expected success to be false for a new pipeline'

    pi.success = false
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Failed, pi.state, 'expected state to be Failed after updating success to false'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false for a new pipeline'

    pi.state = PipelineInstance::RunningOnServer
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::RunningOnServer, pi.state, 'expected state to be RunningOnServer after updating state to RunningOnServer'
    assert pi.active, 'expected active to be true after update'
    assert !pi.success, 'expected success to be alse after update'

    pi.state = PipelineInstance::Paused
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Paused, pi.state, 'expected state to be Paused after updating state to Paused'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false after update'

    pi.state = PipelineInstance::Complete
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Complete, pi.state, 'expected state to be Complete after updating state to Complete'
    assert !pi.active, 'expected active to be false after update'
    assert pi.success, 'expected success to be true after update'

    pi.state = 'bogus'
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Complete, pi.state, 'expected state to be unchanged with set to a bogus value'
    assert !pi.active, 'expected active to be false after update'
    assert pi.success, 'expected success to be true after update'

    pi.state = PipelineInstance::Failed
    pi.save
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::Failed, pi.state, 'expected state to be Failed after updating state to Failed'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false after update'
  end

  test "update attributes for pipeline with two components" do
    pi = pipeline_instances :new_pipeline

    # add two components, one with input and one with no input and expect state to be New
    component1 = {'script_parameters' => {"something" => "xxxad4b39ca5a924e481008009d94e32+210", "input" => "c1bad4b39ca5a924e481008009d94e32+210"}}
    component2 = {'script_parameters' => {"something_else" => "xxxad4b39ca5a924e481008009d94e32+210", "input_missing" => {"required" => true}}}
    pi.components['first'] = component1
    pi.components['second'] = component2
    components = pi.components
    pi.update_attribute 'components', pi.components
    pi = PipelineInstance.find_by_uuid 'zzzzz-xxxxx-f4gneyn6br1xize'
    assert_equal PipelineInstance::New, pi.state, 'expected state to be New after adding component with input'
    assert_equal pi.components.size, 2, 'expected two components'
    assert !pi.active, 'expected active to be false after update'
    assert !pi.success, 'expected success to be false for a new pipeline'
  end

end
