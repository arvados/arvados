# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module Trashable
  def self.included(base)
    base.before_validation :set_validation_timestamp
    base.before_validation :ensure_trash_at_not_in_past
    base.before_validation :sync_trash_state
    base.before_validation :default_trash_interval
    base.validate :validate_trash_and_delete_timing
  end

  # Use a single timestamp for all validations, even though each
  # validation runs at a different time.
  def set_validation_timestamp
    @validation_timestamp = db_current_time
  end

  # If trash_at is being changed to a time in the past, change it to
  # now. This allows clients to say "expires {client-current-time}"
  # without failing due to clock skew, while avoiding odd log entries
  # like "expiry date changed to {1 year ago}".
  def ensure_trash_at_not_in_past
    if trash_at_changed? && trash_at
      self.trash_at = [@validation_timestamp, trash_at].max
    end
  end

  # Caller can move into/out of trash by setting/clearing is_trashed
  # -- however, if the caller also changes trash_at, then any changes
  # to is_trashed are ignored.
  def sync_trash_state
    if is_trashed_changed? && !trash_at_changed?
      if is_trashed
        self.trash_at = @validation_timestamp
      else
        self.trash_at = nil
        self.delete_at = nil
      end
    end
    self.is_trashed = trash_at && trash_at <= @validation_timestamp || false
    true
  end

  def default_delete_after_trash_interval
    Rails.configuration.Collections.DefaultTrashLifetime
  end

  def minimum_delete_after_trash_interval
    Rails.configuration.Collections.BlobSigningTTL
  end

  def default_trash_interval
    if trash_at_changed? && !delete_at_changed?
      # If trash_at is updated without touching delete_at,
      # automatically update delete_at to a sensible value.
      if trash_at.nil?
        self.delete_at = nil
      else
        self.delete_at = trash_at + self.default_delete_after_trash_interval
      end
    elsif !trash_at || !delete_at || trash_at > delete_at
      # Not trash, or bogus arguments? Just validate in
      # validate_trash_and_delete_timing.
    elsif delete_at_changed? && delete_at >= trash_at
      # Fix delete_at if needed, so it's not earlier than the expiry
      # time on any permission tokens that might have been given out.

      # In any case there are no signatures expiring after now+TTL.
      # Also, if the existing trash_at time has already passed, we
      # know we haven't given out any signatures since then.
      earliest_delete = [
        @validation_timestamp,
        trash_at_was,
      ].compact.min + minimum_delete_after_trash_interval

      # The previous value of delete_at is also an upper bound on the
      # longest-lived permission token. For example, if TTL=14,
      # trash_at_was=now-7, delete_at_was=now+7, then it is safe to
      # set trash_at=now+6, delete_at=now+8.
      earliest_delete = [earliest_delete, delete_at_was].compact.min

      # If delete_at is too soon, use the earliest possible time.
      if delete_at < earliest_delete
        self.delete_at = earliest_delete
      end
    end
  end

  def validate_trash_and_delete_timing
    if trash_at.nil? != delete_at.nil?
      errors.add :delete_at, "must be set if trash_at is set, and must be nil otherwise"
    elsif delete_at && delete_at < trash_at
      errors.add :delete_at, "must not be earlier than trash_at"
    end
    true
  end
end

module TrashableController
  def self.included(base)
    def base._trash_method_description
      match = name.match(/\b(\w+)Controller$/)
      "Trash a #{match[1].singularize.underscore.humanize.downcase}."
    end
    def base._untrash_method_description
      match = name.match(/\b(\w+)Controller$/)
      "Untrash a #{match[1].singularize.underscore.humanize.downcase}."
    end
  end

  def destroy
    if !@object.is_trashed
      @object.update!(trash_at: db_current_time)
    end
    earliest_delete = (@object.trash_at + @object.minimum_delete_after_trash_interval)
    if @object.delete_at > earliest_delete
      @object.update!(delete_at: earliest_delete)
    end
    show
  end

  def trash
    if !@object.is_trashed
      @object.update!(trash_at: db_current_time)
    end
    show
  end

  def untrash
    if !@object.is_trashed
      raise ArvadosModel::InvalidStateTransitionError.new("Item is not trashed, cannot untrash")
    end

    if db_current_time >= @object.delete_at
      raise ArvadosModel::InvalidStateTransitionError.new("delete_at time has already passed, cannot untrash")
    end

    @object.trash_at = nil

    if params[:ensure_unique_name]
      @object.save_with_unique_name!
    else
      @object.save!
    end

    show
  end
end
