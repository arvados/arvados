require 'test_helper'

class CollectionsHelperTest < ActionView::TestCase
  reset_api_fixtures :after_each_test, false

  [
    ["filename.csv", true],
    ["filename.fa", true],
    ["filename.fasta", true],
    ["filename.seq", true],   # another fasta extension
    ["filename.go", true],
    ["filename.htm", true],
    ["filename.html", true],
    ["filename.json", true],
    ["filename.md", true],
    ["filename.pdf", true],
    ["filename.py", true],
    ["filename.R", true],
    ["filename.sam", true],
    ["filename.sh", true],
    ["filename.txt", true],
    ["filename.tiff", true],
    ["filename.tsv", true],
    ["filename.vcf", true],
    ["filename.xml", true],
    ["filename.xsl", true],
    ["filename.yml", true],

    ["filename.bam", false],
    ["filename", false],
  ].each do |file_name, preview_allowed|
    test "verify '#{file_name}' is allowed for preview #{preview_allowed}" do
      assert_equal preview_allowed, preview_allowed_for(file_name)
    end
  end
end
