module DownloadHelper
  module_function

  def path
    Rails.root.join 'tmp', 'downloads'
  end

  def clear
    if File.exist? path
      FileUtils.rm_r path
    end
    begin
      Dir.mkdir path
    rescue Errno::EEXIST
    end
  end

  def done
    Dir[path.join '*'].reject do |f|
      /\.part$/ =~ f
    end
  end
end
