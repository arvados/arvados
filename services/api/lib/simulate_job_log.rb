module SimulateJobLog
	def replay(filename, multiplier = 1, simulated_job_uuid = nil)
		raise "Environment must be development or test" unless [ 'test', 'development' ].include? ENV['RAILS_ENV']

	    multiplier = multiplier.to_f
	    multiplier = 1.0 if multiplier <= 0

	    actual_start_time = Time.now
	    log_start_time = nil

		act_as_system_user do
			File.open(filename).each.with_index do |line, index|
				cols = {}
		        cols[:timestamp], cols[:job_uuid], cols[:pid], cols[:task], cols[:event_type], cols[:message] = line.split(' ', 6)
		        cols[:timestamp] = Time.strptime( cols[:timestamp], "%Y-%m-%d_%H:%M:%S" )
		        # Override job uuid with a simulated one if specified
		        cols[:job_uuid] = simulated_job_uuid || cols[:job_uuid]
		        # determine when we want to simulate this log being created, based on the time multiplier
		        log_start_time = cols[:timestamp] if log_start_time.nil?
		        log_time = cols[:timestamp]
		        actual_elapsed_time = Time.now - actual_start_time
		        log_elapsed_time = log_time - log_start_time
	            modified_elapsed_time = log_elapsed_time / multiplier
		        pause_time = modified_elapsed_time - actual_elapsed_time
		        if pause_time > 0
			        sleep pause_time
			    end
			    # output log entry for debugging and create it in the current environment's database
		        puts "#{index} #{cols.to_yaml}\n"
		        Log.new({
		        	event_at:    Time.zone.local_to_utc(cols[:timestamp]),
		        	object_uuid: cols[:job_uuid],
		        	event_type:  cols[:event_type],
		        	properties:  { 'text' => line }
		        }).save!
			end
		end

	end
end
