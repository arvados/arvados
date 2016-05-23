class WorkUnit
  # This is just an abstract class that documents the WorkUnit interface; a
  # class can implement the interface without being a subclass of WorkUnit.

  def label
    # returns the label that was assigned when creating the work unit
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

  def created_at
    # returns created_at timestamp
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

  def state_bootstrap_class
    # returns a class like "danger", "success", or "warning" that a view can use directly to make a display class
  end

  def success?
    # returnis true if the work unit finished successfully,
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

  def output
    # returns uuid or pdh of output data, if any
  end

  def can_cancel?
    # returns if this work unit is cancelable
  end

  def uri
    # returns the uri for this work unit
  end

  def child_summary
    # summary status of any children of this work unit
  end

  def title
    # title for the work unit
  end
end
