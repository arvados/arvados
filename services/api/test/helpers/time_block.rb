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
end
