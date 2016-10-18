class WorkUnit
  # This is an abstract class that documents the WorkUnit interface

  def label
    # returns the label that was assigned when creating the work unit
  end

  def proxied
    # returns the proxied object of this work unit
  end

  def uuid
    # returns the arvados UUID of the underlying object
  end

  def children
    # returns an array of child work units
  end

  def modified_by_user_uuid
    # returns uuid of the user who modified this work unit most recently
  end

  def owner_uuid
    # returns uuid of the owner of this work unit
  end

  def created_at
    # returns created_at timestamp
  end

  def modified_at
    # returns modified_at timestamp
  end

  def started_at
    # returns started_at timestamp for this work unit
  end

  def finished_at
    # returns finished_at timestamp
  end

  def state_label
    # returns a string representing state of the work unit
  end

  def exit_code
    # returns the work unit's execution exit code
  end

  def state_bootstrap_class
    # returns a class like "danger", "success", or "warning" that a view can use directly to make a display class
  end

  def success?
    # returns true if the work unit finished successfully,
    # false if it has a permanent failure,
    # and nil if the final state is not determined.
  end

  def progress
    # returns a number between 0 and 1
  end

  def log_collection
    # returns uuid or pdh with saved log data, if any
  end

  def parameters
    # returns work unit parameters, if any
  end

  def script
    # returns script for this work unit, if any
  end

  def repository
    # returns this work unit's script repository, if any
  end

  def script_version
    # returns this work unit's script_version, if any
  end

  def supplied_script_version
    # returns this work unit's supplied_script_version, if any
  end

  def docker_image
    # returns this work unit's docker_image, if any
  end

  def runtime_constraints
    # returns this work unit's runtime_constraints, if any
  end

  def priority
    # returns this work unit's priority, if any
  end

  def nondeterministic
    # returns if this is nondeterministic
  end

  def outputs
    # returns array containing uuid or pdh of output data
  end

  def child_summary
    # summary status of any children of this work unit
  end

  def child_summary_str
    # textual representation of child summary
  end

  def can_cancel?
    # returns true if this work unit can be canceled
  end

  def uri
    # returns the uri for this work unit
  end

  def title
    # title for the work unit
  end

  def has_unreadable_children
    # accept it if you can't understand your own children
  end

  # view helper methods
  def walltime
    # return walltime for a running or completed work unit
  end

  def cputime
    # return cputime for a running or completed work unit
  end

  def queuedtime
    # return queued time if the work unit is queued
  end

  def is_running?
    # is the work unit in running state?
  end

  def is_paused?
    # is the work unit in paused state?
  end

  def is_finished?
    # is the work unit in finished state?
  end

  def is_failed?
    # is this work unit in failed state?
  end

  def command
    # command to execute
  end

  def cwd
    # initial workind directory
  end

  def environment
    # environment variables
  end

  def mounts
    # mounts
  end

  def output_path
    # path to a directory or file to save output
  end

  def container_uuid
    # container_uuid of a container_request
  end

  def log_object_uuids
    # object uuids for live log
  end

  def live_log_lines(limit)
    # fetch log entries from logs table for @proxied
  end

  def render_log
    # return partial and locals to be rendered
  end

  def template_uuid
    # return the uuid of this work unit's template, if one exists
  end
end
