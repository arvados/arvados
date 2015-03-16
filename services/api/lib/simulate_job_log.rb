require 'db_current_time'

module SimulateJobLog
  include DbCurrentTime

  def replay(filename, multiplier = 1, simulated_job_uuid = nil)
    raise "Environment must be development or test" unless [ 'test', 'development' ].include? ENV['RAILS_ENV']

    multiplier = multiplier.to_f
    multiplier = 1.0 if multiplier <= 0

    actual_start_time = db_current_time
    log_start_time = nil

    act_as_system_user do
      File.open(filename).each.with_index do |line, index|
        cols = {}
        cols[:timestamp], rest_of_line = line.split(' ', 2)
        begin
          cols[:timestamp] = Time.strptime( cols[:timestamp], "%Y-%m-%d_%H:%M:%S" )
        rescue ArgumentError
          if line =~ /^((?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{1,2} \d\d:\d\d:\d\d \d{4}) (.*)/
            # Wed Nov 19 07:12:39 2014
            cols[:timestamp] = Time.strptime( $1, "%a %b %d %H:%M:%S %Y" )
            rest_of_line = $2
          else
              STDERR.puts "Ignoring log line because of unknown time format: #{line}"
          end
        end
        cols[:job_uuid], cols[:pid], cols[:task], cols[:event_type], cols[:message] = rest_of_line.split(' ', 5)
        # Override job uuid with a simulated one if specified
        cols[:job_uuid] = simulated_job_uuid || cols[:job_uuid]
        # determine when we want to simulate this log being created, based on the time multiplier
        log_start_time = cols[:timestamp] if log_start_time.nil?
        log_time = cols[:timestamp]
        actual_elapsed_time = db_current_time - actual_start_time
        log_elapsed_time = log_time - log_start_time
        modified_elapsed_time = log_elapsed_time / multiplier
        pause_time = modified_elapsed_time - actual_elapsed_time
        sleep pause_time if pause_time > 0
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
