class NodesController < ApplicationController
  def index
    @objects = model_class.order("created_at desc")

    @slurm_state = {}
    IO.popen('sinfo --noheader --Node || echo "compute[1-3] foo bar DOWN"').readlines.each do |line|
      tokens = line.strip.split
      nodestate = tokens.last
      nodenames = []
      if (re = tokens.first.match /^([^\[]*)\[([-\d,]+)\]$/)
        nodeprefix = re[1]
        re[2].split(',').each do |number_range|
          if number_range.index('-')
            range = number_range.split('-').collect(&:to_i)
            (range[0]..range[1]).each do |n|
              nodenames << "#{nodeprefix}#{n}"
            end
          else
            nodenames << "#{nodeprefix}#{number_range}"
          end
        end
      else
        nodenames << tokens.first
      end
      nodenames.each do |nodename|
        @slurm_state[nodename] = nodestate.downcase
      end
    end
  end
end
