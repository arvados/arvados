class PopulateScriptParametersDigest < ActiveRecord::Migration
  def up
    done = false
    while !done
      done = true
      Job.
        where('script_parameters_digest is null').
        select([:id, :script_parameters, :script_parameters_digest]).
        limit(200).
        each do |j|
        done = false
        Job.
          where('id=? or script_parameters=?', j.id, j.script_parameters.to_yaml).
          update_all(script_parameters_digest: j.update_script_parameters_digest)
      end
    end
  end

  def down
  end
end
