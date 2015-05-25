module ManifestExamples
  def make_manifest opts={}
    opts = {
      bytes_per_block: 1,
      blocks_per_file: 1,
      files_per_stream: 1,
      streams: 1,
    }.merge(opts)
    datablip = "x" * opts[:bytes_per_block]
    locator = Blob.sign_locator(Digest::MD5.hexdigest(datablip) +
                                '+' + datablip.length.to_s,
                                api_token: opts[:api_token])
    filesize = datablip.length * opts[:blocks_per_file]
    txt = ''
    (1..opts[:streams]).each do |s|
      streamtoken = "./stream#{s}"
      streamsize = 0
      blocktokens = []
      filetokens = []
      (1..opts[:files_per_stream]).each do |f|
        filetokens << " #{streamsize}:#{filesize}:file#{f}.txt"
        (1..opts[:blocks_per_file]).each do |b|
          blocktokens << locator
        end
        streamsize += filesize
      end
      txt << ([streamtoken] + blocktokens + filetokens).join(' ') + "\n"
    end
    txt
  end
end
