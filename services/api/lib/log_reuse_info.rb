module LogReuseInfo
  # log_reuse_info logs whatever the given block returns, if
  # log_reuse_decisions is enabled. It accepts a block instead of a
  # string because in some cases constructing the strings involves
  # doing database queries, and we want to skip those queries when
  # logging is disabled.
  def log_reuse_info
    if Rails.configuration.log_reuse_decisions
      Rails.logger.info("find_reusable: " + yield)
    end
  end
end
