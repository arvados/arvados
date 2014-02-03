class PipelineInstancesController < ApplicationController
  skip_before_filter :find_object_by_uuid, only: :compare
  before_filter :find_objects_by_uuid, only: :compare
  include PipelineInstancesHelper

  def compare
    @breadcrumb_page_name = 'compare'

    @rows = []          # each is {name: S, components: [...]}

    # Build a table: x=pipeline y=component
    @objects.each_with_index do |pi, pi_index|
      pipeline_jobs(pi).each do |component|
        # Find a cell with the same name as this component but no
        # entry for this pipeline
        target_row = nil
        @rows.each_with_index do |row, row_index|
          if row[:name] == component[:name] and !row[:components][pi_index]
            target_row = row
          end
        end
        if !target_row
          target_row = {name: component[:name], components: []}
          @rows << target_row
        end
        target_row[:components][pi_index] = component
      end
    end

    @rows.each do |row|
      # Build a "normal" pseudo-component for this row by picking the
      # most common value for each attribute. If all values are
      # equally common, there is no "normal".
      normal = {}              # attr => most common value
      highscore = {}           # attr => how common "normal" is
      score = {}               # attr => { value => how common }
      row[:components].each do |pj|
        pj.each do |k,v|
          vstr = for_comparison v
          score[k] ||= {}
          score[k][vstr] = (score[k][vstr.to_s] || 0) + 1
          highscore[k] ||= 0
          if score[k][vstr] == highscore[k]
            # tie for first place = no "normal"
            normal.delete k
          elsif score[k][vstr] == highscore[k] + 1
            # more pipelines have v than anything else
            highscore[k] = score[k][vstr]
            normal[k] = vstr
          end
        end
      end

      # Add a hash in component[:is_normal]: { attr => is_the_value_normal? }
      row[:components].each do |pj|
        pj[:is_normal] = {}
        pj.each do |k,v|
          pj[:is_normal][k] = (normal.has_key?(k) && normal[k] == for_comparison(v))
        end
      end
    end
  end

  protected
  def for_comparison v
    if v.is_a? Hash or v.is_a? Array
      v.to_json
    else
      v.to_s
    end
  end

  def find_objects_by_uuid
    @objects = model_class.where(uuid: params[:uuids])
  end

end
