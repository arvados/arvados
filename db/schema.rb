# encoding: UTF-8
# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# Note that this schema.rb definition is the authoritative source for your
# database schema. If you need to create the application database on another
# system, you should be using db:schema:load, not running all the migrations
# from scratch. The latter is a flawed and unsustainable approach (the more migrations
# you'll amass, the slower it'll run and the greater likelihood for issues).
#
# It's strongly recommended to check this file into your version control system.

ActiveRecord::Schema.define(:version => 20130105224618) do

  create_table "collections", :force => true do |t|
    t.string   "locator"
    t.string   "created_by_client"
    t.string   "created_by_user"
    t.datetime "created_at"
    t.string   "modified_by_client"
    t.string   "modified_by_user"
    t.datetime "modified_at"
    t.string   "portable_data_hash"
    t.string   "name"
    t.integer  "redundancy"
    t.string   "redundancy_confirmed_by_client"
    t.datetime "redundancy_confirmed_at"
    t.integer  "redundancy_confirmed_as"
    t.datetime "updated_at"
  end

  create_table "metadata", :force => true do |t|
    t.string   "uuid"
    t.string   "created_by_client"
    t.string   "created_by_user"
    t.datetime "created_at"
    t.string   "modified_by_client"
    t.string   "modified_by_user"
    t.datetime "modified_at"
    t.string   "target_uuid"
    t.string   "target_kind"
    t.integer  "native_target_id"
    t.string   "native_target_type"
    t.string   "metadata_class"
    t.string   "key"
    t.string   "value"
    t.text     "info"
    t.datetime "updated_at"
  end

end
