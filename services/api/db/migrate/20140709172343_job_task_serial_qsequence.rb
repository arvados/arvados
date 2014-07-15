class JobTaskSerialQsequence < ActiveRecord::Migration
  SEQ_NAME = "job_tasks_qsequence_seq"

  def up
    execute "CREATE SEQUENCE #{SEQ_NAME} OWNED BY job_tasks.qsequence;"
  end

  def down
    execute "DROP SEQUENCE #{SEQ_NAME};"
  end
end
