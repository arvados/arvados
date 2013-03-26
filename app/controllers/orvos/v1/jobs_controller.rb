class Orvos::V1::JobsController < ApplicationController
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :resource_limits, Hash
  accept_attribute_as_json :tasks_summary, Hash

  def index
    want_ancestor = @where[:script_version_descends_from]
    if want_ancestor
      # Check for missing commit_ancestor rows, and create them if
      # possible.
      @objects.
        dup.
        includes(:commit_ancestors). # I wish Rails would let me
                                     # specify here which
                                     # commit_ancestors I am
                                     # interested in.
        each do |o|
        if o.commit_ancestors.
            select { |ca| ca.ancestor == want_ancestor }.
            empty? and !o.script_version.nil?
          begin
            o.commit_ancestors << CommitAncestor.find_or_create_by_descendant_and_ancestor(o.script_version, want_ancestor)
          rescue
          end
        end
        o.commit_ancestors.
          select { |ca| ca.ancestor == want_ancestor }.
          select(&:is).
          first
      end
      # Now it is safe to do an .includes().where() because we are no
      # longer interested in jobs that have other ancestors but not
      # want_ancestor.
      @objects = @objects.
        includes(:commit_ancestors).
        where('commit_ancestors.ancestor = ? and commit_ancestors.is = ?',
              want_ancestor, true)
    end
    super
  end
end
