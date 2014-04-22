# Expects:
#   +params+ Hash
# Sets:
#   @where, @filters

module LoadParam

  def load_where_param
    if params[:where].nil? or params[:where] == ""
      @where = {}
    elsif params[:where].is_a? Hash
      @where = params[:where]
    elsif params[:where].is_a? String
      begin
        @where = Oj.load(params[:where])
        raise unless @where.is_a? Hash
      rescue
        raise ArgumentError.new("Could not parse \"where\" param as an object")
      end
    end
    @where = @where.with_indifferent_access
  end

  def load_filters_param
    @filters ||= []
    if params[:filters].is_a? Array
      @filters += params[:filters]
    elsif params[:filters].is_a? String and !params[:filters].empty?
      begin
        f = Oj.load params[:filters]
        raise unless f.is_a? Array
        @filters += f
      rescue
        raise ArgumentError.new("Could not parse \"filters\" param as an array")
      end
    end
  end

end
