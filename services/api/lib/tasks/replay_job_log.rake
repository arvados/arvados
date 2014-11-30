require 'simulate_job_log'
desc 'Simulate job logging from a file. Three arguments: log filename, time multipler (optional), simulated job uuid (optional). E.g. (use quotation marks if using spaces between args): rake "replay_job_log[log.txt, 2.0, qr1hi-8i9sb-nf3qk0xzwwz3lre]"'
task :replay_job_log, [:filename, :multiplier, :uuid] => :environment do |t, args|
  include SimulateJobLog
  abort("No filename specified.") if args[:filename].blank?
  replay( args[:filename], args[:multiplier].to_f, args[:uuid] )
end
