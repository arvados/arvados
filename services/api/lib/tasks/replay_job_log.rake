require 'simulate_job_log'
desc 'Simulate job logging from a file. Three arguments: log filename, simulated job uuid (optional), time multipler (optional). E.g. (note quotation marks): rake "replay_job_log[log.txt, qr1hi-8i9sb-nf3qk0xzwwz3lre, 2.0]"'
task :replay_job_log, [:filename, :uuid, :multiplier] => :environment do |t, args|
	include SimulateJobLog
    abort("No filename specified.") if args[:filename].blank?
    replay( args[:filename], args[:uuid], args[:multiplier].to_f )
end
