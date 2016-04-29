class ActiveSupport::TestCase
  def time_block label
    t0 = Time.now
    begin
      yield
    ensure
      t1 = Time.now
      $stderr.puts "#{t1 - t0}s #{label}"
    end
  end

  def vmpeak c
    open("/proc/self/status").each_line do |line|
      print "Begin #{c} #{line}" if (line =~ /^VmHWM:/)
    end
    n = yield
    open("/proc/self/status").each_line do |line|
      print "End #{c} #{line}" if (line =~ /^VmHWM:/)
    end
    n
  end

end
