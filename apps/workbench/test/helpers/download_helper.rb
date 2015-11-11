module DownloadHelper
  module_function

  def path
    Rails.root.join 'tmp', 'downloads'
  end

  def clear
    FileUtils.rm_f path
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
