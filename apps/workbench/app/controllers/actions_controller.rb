# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require "arvados/collection"
require "app_version"

class ActionsController < ApplicationController

  # Skip require_thread_api_token if this is a show action
  # for an object uuid that supports anonymous access.
  skip_around_action :require_thread_api_token, if: proc { |ctrl|
    !Rails.configuration.Users.AnonymousUserToken.empty? and
    'show' == ctrl.action_name and
    params['uuid'] and
    model_class.in?([Collection, Group, Job, PipelineInstance, PipelineTemplate])
  }
  skip_around_action :require_thread_api_token, only: [:report_issue_popup, :report_issue]
  skip_before_action :check_user_agreements, only: [:report_issue_popup, :report_issue]

  @@exposed_actions = {}
  def self.expose_action method, &block
    @@exposed_actions[method] = true
    define_method method, block
  end

  def model_class
    ArvadosBase::resource_class_for_uuid(params[:uuid])
  end

  def show
    @object = model_class.andand.find(params[:uuid])
    if @object.is_a? Link and
        @object.link_class == 'name' and
        ArvadosBase::resource_class_for_uuid(@object.head_uuid) == Collection
      redirect_to collection_path(id: @object.uuid)
    elsif @object.is_a?(Group) and (@object.group_class == 'project' or @object.group_class == 'filter')
      redirect_to project_path(id: @object.uuid)
    elsif @object
      redirect_to @object
    else
      raise ActiveRecord::RecordNotFound
    end
  end

  def post
    params.keys.collect(&:to_sym).each do |param|
      if @@exposed_actions[param]
        return self.send(param)
      end
    end
    redirect_back(fallback_location: root_path)
  end

  expose_action :copy_selections_into_project do
    move_or_copy :copy
  end

  expose_action :move_selections_into_project do
    move_or_copy :move
  end

  def move_or_copy action
    uuids_to_add = params["selection"]
    uuids_to_add = [ uuids_to_add ] unless uuids_to_add.is_a? Array
    resource_classes = uuids_to_add.
      collect { |x| ArvadosBase::resource_class_for_uuid(x) }.
      uniq
    resource_classes.each do |resource_class|
      resource_class.filter([['uuid','in',uuids_to_add]]).each do |src|
        if resource_class == Collection and not Collection.attribute_info.include?(:name)
          dst = Link.new(owner_uuid: @object.uuid,
                         tail_uuid: @object.uuid,
                         head_uuid: src.uuid,
                         link_class: 'name',
                         name: src.uuid)
        else
          case action
          when :copy
            dst = src.dup
            if dst.respond_to? :'name='
              if dst.name
                dst.name = "Copy of #{dst.name}"
              else
                dst.name = "Copy of unnamed #{dst.class_for_display.downcase}"
              end
            end
            if resource_class == Collection
              dst.manifest_text = Collection.select([:manifest_text]).where(uuid: src.uuid).with_count("none").first.manifest_text
              # Fixes bug 19144: nullify some fields that are managed by keep-balance.
              dst.storage_classes_confirmed = []
              dst.storage_classes_confirmed_at = nil
            end
          when :move
            dst = src
          else
            raise ArgumentError.new "Unsupported action #{action}"
          end
          dst.owner_uuid = @object.uuid
          dst.tail_uuid = @object.uuid if dst.class == Link
        end
        begin
          dst.save!
        rescue
          dst.name += " (#{Time.now.localtime})" if dst.respond_to? :name=
          dst.save!
        end
      end
    end
    if (resource_classes == [Collection] and
        @object.is_a? Group and
        @object.group_class == 'project') or
        @object.is_a? User
      # In the common case where only collections are copied/moved
      # into a project, it's polite to land on the collections tab on
      # the destination project.
      redirect_to project_url(@object.uuid, anchor: 'Data_collections')
    else
      # Otherwise just land on the default (Description) tab.
      redirect_to @object
    end
  end

  expose_action :combine_selected_files_into_collection do
    uuids, source_paths = selected_collection_files params

    new_coll = Arv::Collection.new
    Collection.where(uuid: uuids.uniq).with_count("none").
        select([:uuid, :manifest_text]).each do |coll|
      src_coll = Arv::Collection.new(coll.manifest_text)
      src_pathlist = source_paths[coll.uuid]
      if src_pathlist.any?(&:blank?)
        src_pathlist = src_coll.each_file_path
        destdir = nil
      else
        destdir = "."
      end
      src_pathlist.each do |src_path|
        src_path = src_path.sub(/^(\.\/|\/|)/, "./")
        src_stream, _, basename = src_path.rpartition("/")
        dst_stream = destdir || src_stream
        # Generate a unique name by adding (1), (2), etc. to it.
        # If the filename has a dot that's not at the beginning, insert the
        # number just before that.  Otherwise, append the number to the name.
        if match = basename.match(/[^\.]\./)
          suffix_start = match.begin(0) + 1
        else
          suffix_start = basename.size
        end
        suffix_size = 0
        dst_path = nil
        loop.each_with_index do |_, try_count|
          dst_path = "#{dst_stream}/#{basename}"
          break unless new_coll.exist?(dst_path)
          uniq_suffix = "(#{try_count + 1})"
          basename[suffix_start, suffix_size] = uniq_suffix
          suffix_size = uniq_suffix.size
        end
        new_coll.cp_r(src_path, dst_path, src_coll)
      end
    end

    coll_attrs = {
      manifest_text: new_coll.manifest_text,
      name: "Collection created at #{Time.now.localtime}",
    }
    flash = {}

    # set owner_uuid to current project, provided it is writable
    action_data = Oj.load(params['action_data'] || "{}")
    if action_data['current_project_uuid'] and
        current_project = Group.find?(action_data['current_project_uuid']) and
        current_project.writable_by.andand.include?(current_user.uuid)
      coll_attrs[:owner_uuid] = current_project.uuid
      flash[:message] =
        "Created new collection in the project #{current_project.name}."
    else
      flash[:message] = "Created new collection in your Home project."
    end

    newc = Collection.create!(coll_attrs)
    source_paths.each_key do |src_uuid|
      unless Link.create({
                           tail_uuid: src_uuid,
                           head_uuid: newc.uuid,
                           link_class: "provenance",
                           name: "provided",
                         })
        flash[:error] = "
An error occurred when saving provenance information for this collection.
You can try recreating the collection to get a copy with full provenance data."
        break
      end
    end
    redirect_to(newc, flash: flash)
  end

  def report_issue_popup
    respond_to do |format|
      format.js
      format.html
    end
  end

  def report_issue
    logger.warn "report_issue: #{params.inspect}"

    respond_to do |format|
      IssueReporter.send_report(current_user, params).deliver
      format.js {render body: nil}
    end
  end

  # star / unstar the current project
  def star
    links = Link.where(owner_uuid: current_user.uuid,
                       head_uuid: @object.uuid,
                       link_class: 'star')

    if params['status'] == 'create'
      # create 'star' link if one does not already exist
      if !links.andand.any?
        dst = Link.new(owner_uuid: current_user.uuid,
                       tail_uuid: current_user.uuid,
                       head_uuid: @object.uuid,
                       link_class: 'star',
                       name: @object.uuid)
        dst.save!
      end
    else # delete any existing 'star' links
      if links.andand.any?
        links.each do |link|
          link.destroy
        end
      end
    end

    respond_to do |format|
      format.js
    end
  end

  protected

  def derive_unique_filename filename, manifest_files
    filename_parts = filename.split('.')
    filename_part = filename_parts[0]
    counter = 1
    while true
      return filename if !manifest_files.include? filename
      filename_parts[0] = filename_part + "(" + counter.to_s + ")"
      filename = filename_parts.join('.')
      counter += 1
    end
  end

end
