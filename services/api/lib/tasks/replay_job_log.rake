require 'simulate_job_log'
desc 'Simulate job logging from a file. Four arguments: log filename, time multipler (optional), delete existing log entries (optional, default is false), simulated job uuid (optional). Note that deleting existing log entries only works if a simulated job uuid is also specified. E.g. (use quotation marks if using spaces between args): rake "replay_job_log[log.txt, 2.0, true, qr1hi-8i9sb-nf3qk0xzwwz3lre]"'
task :replay_job_log, [:filename, :multiplier, :delete_log_entries, :uuid] => :environment do |t, args|
	include SimulateJobLog
    abort("No filename specified.") if args[:filename].blank?
    replay( args[:filename], args[:multiplier].to_f, args[:delete_log_entries], args[:uuid] )
end
