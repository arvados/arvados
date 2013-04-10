module VcfPipelineHelper
  require 'csv'

  def reset_vcf_pipeline_instance(pi, input_manifest)
    params = {
      'PICARD_ZIP' => '7a4073e29bfa87154b7102e75668c454+83+K@van',
      'GATK_BUNDLE' => '0a37aaf212464efa2a77ff9ba51c0148+10524+K@van',
      'GATK_TAR_BZ2' => '482ebab0408e173370c499f0b7c00878+93+K@van',
      'BWA' => '73be5598809c66f260fedd253c8608bd+67+K@van',
      'SAM' => '55d2115faa608eb95dab4f875b7511b1+72+K@van',
      'REGION_PADDING' => '10',
      'REGIONS' => 'e52c086f41c2f089d88ec2bbd45355d3+87+K@van/SeqCap_EZ_Exome_v2.hg19.bed',
      'STAND_CALL_CONF' => '4.0',
      'STAND_EMIT_CONF' => '4.0',
      "bwa/INPUT" => input_manifest
    }
    pi.components = PipelineTemplate.find(pi.pipeline_uuid).components
    pi.update_job_parameters(params)
    pi.active = true
    pi.success = nil
  end

  def vcf_pipeline_summary(pi)
    stats = {}
    collection_link = Link.
      where(head_uuid: pi.uuid,
            link_class: 'client-defined',
            name: 'vcffarm-pipeline-invocation').
      last
    if collection_link
      stats[:collection_uuid] = collection_link.tail_uuid
    else
      pi.components[:steps].each do |step|
        if step[:name] == 'bwa'
          step[:params].each do |param|
            if param[:name] == 'INPUT'
              stats[:collection_uuid] = param[:data_locator] || param[:value]
              break
            end
          end
        end
      end
    end
    if stats[:collection_uuid]
      Link.where(tail_uuid: stats[:collection_uuid],
                 head_kind: Group)[0..0].each do |c2p|
        stats[:project_uuid] = c2p.head_uuid
        group = Group.find stats[:project_uuid]
        stats[:project_name] = group.name rescue nil
      end
      Link.where(tail_uuid: stats[:collection_uuid],
                 head_kind: Specimen)[0..0].each do |c2s|
        stats[:specimen_uuid] = c2s.head_uuid
        specimen = Specimen.find stats[:specimen_uuid]
        stats[:specimen_id] = specimen.properties[:specimen_id] rescue nil
      end
    end
    stats[:runtime] = {}
    stats[:alignment_for_step] = {}
    stats[:alignment] = {}
    stats[:coverage] = []
    pi.components[:steps].each do |step|
      if step[:warehousejob]
        if step[:name] == 'bwa' and step[:warehousejob][:starttime]
          stats[:runtime][:started_at] = step[:warehousejob][:starttime]
        end
        if step[:warehousejob][:finishtime]
          stats[:runtime][:finished_at] =
            [ step[:warehousejob][:finishtime],
              stats[:runtime][:finished_at] ].compact.max
        end
      end
      if step[:name] == 'picard-casm' and
          step[:complete] and
          step[:output_data_locator]
        tsv = IO.
          popen("whget -r #{step[:output_data_locator]}/ -").
          readlines.
          collect { |x| x.strip.split "\t" }
        casm = {}
        head = []
        tsv.each do |data|
          if data.size < 4 or data[0].match /^\#/
            next
          elsif data[0] == 'CATEGORY' or data[1].match /[^\d\.]/
            head = data
          elsif data[0] == 'PAIR'
            head.each_with_index do |name, index|
              x = data[index]
              if x and x.match /^\d+$/
                x = x.to_i
              elsif x and x.match /^\d+\.\d+$/
                x = x.to_f
              end
              name = name.downcase.to_sym
              casm[name] ||= []
              casm[name] << x
            end
          end
        end
        stats[:picard_alignment_summary] = casm
      end
      if step[:name] == 'gatk-stats' and
          step[:complete] and
          step[:output_data_locator]
        csv = IO.
          popen("whget #{step[:output_data_locator]}/mincoverage_nlocus.csv").
          readlines.
          collect { |x| x.strip.split ',' }
        csv.each do |depth, nlocus, percent|
          stats[:coverage][depth.to_i] = nlocus.to_i
        end
      end
      if step[:name] == 'gatk-realign' and
          step[:complete] and
          step[:output_data_locator]
        logs = IO.
          popen("whget #{step[:warehousejob][:metakey]}").
          readlines.
          collect(&:strip)
        logs.each do |logline|
          if (re = logline.match /\s(\d+) stderr INFO .* (\d+) reads were filtered out.*of (\d+) total/)
            stats[:alignment_for_step][re[1]] ||= {}
            stats[:alignment_for_step][re[1]][:filtered_reads] = re[2].to_i
            stats[:alignment_for_step][re[1]][:total_reads] = re[3].to_i
          elsif (re = logline.match /(\d+) reads.* failing BadMate/)
            stats[:alignment][:bad_mate_reads] = re[1].to_i
          elsif (re = logline.match /(\d+) reads.* failing MappingQualityZero/)
            stats[:alignment][:mapq0_reads] = re[1].to_i
          end
        end
      end
      if step[:name] == 'gatk-merge-call' and
          step[:complete] and
          step[:output_data_locator]
        stats[:vcf_file_name] = "#{stats[:project_name]}-#{stats[:specimen_id]}-#{step[:output_data_locator][0..31]}.vcf"
        logs = IO.
          popen("whget #{step[:warehousejob][:metakey]}").
          readlines.
          collect(&:strip)
        logs.each do |logline|
          if (re = logline.match /(\d+) reads were filtered out.*of (\d+) total/)
            stats[:alignment][:filtered_reads] = re[1].to_i
            stats[:alignment][:total_realigned_reads] = re[2].to_i
          elsif (re = logline.match /(\d+) reads.* failing BadMate/)
            stats[:alignment][:bad_mate_reads] = re[1].to_i
          elsif (re = logline.match /(\d+) reads.* failing UnmappedRead/)
            stats[:alignment][:unmapped_reads] = re[1].to_i
          end
        end

        stats[:chromosome_calls] = {}
        tsv = IO.
          popen("whget #{step[:output_data_locator]}/merged.vcf | egrep -v '^#' | cut -f1 | uniq -c").
          readlines.
          collect { |x| x.strip.split }
        tsv.each do |n_variants, sequence_name|
          stats[:chromosome_calls][sequence_name] = n_variants.to_i
        end

        stats[:inferred_sex] = false
        calls = stats[:chromosome_calls]
        if calls['X'] and calls['X'] > 200
          if !calls['Y']
            stats[:inferred_sex] = 'female'
          elsif calls['Y'] * 60 < calls['X']
            # if Y < X/60 they are presumed to be misalignments
            stats[:inferred_sex] = 'female'
          elsif calls['Y'] * 25 > calls['X']
            # if Y > X/25 we presume a Y chromosome was present
            stats[:inferred_sex] = 'male'
          end
        end
      end
    end
    stats[:alignment][:total_reads] = 0
    stats[:alignment][:filtered_reads] ||= 0
    stats[:alignment][:bad_mate_reads] ||= 0
    stats[:alignment][:mapq0_reads] ||= 0
    stats[:alignment_for_step].values.each do |a4s|
      stats[:alignment][:total_reads] += (a4s[:total_reads] || 0)
      stats[:alignment][:filtered_reads] += (a4s[:filtered_reads] || 0)
      stats[:alignment][:bad_mate_reads] += (a4s[:bad_mate_reads] || 0)
      stats[:alignment][:mapq0_reads] += (a4s[:mapq0_reads] || 0)
    end

    if stats[:collection_uuid]
      csv = CSV.parse IO.
        popen("whget #{stats[:collection_uuid]}/SampleSheet.csv -").
        read
      if !csv.empty?
        pivoted = []
        csv[0].each_with_index do |head, col|
          pivoted << csv.collect { |row| row[col] }
        end
        stats[:source_data_csv_columns] = pivoted
      end
    end

    picardas = stats[:picard_alignment_summary]
    stats[:summary_csv_columns] =
      [['PROJECT', stats[:project_name]],
       ['SPECIMEN', stats[:specimen_id]],
       ['VCF_FILE_NAME', stats[:vcf_file_name]],
       ['INFERRED_SEX', stats[:inferred_sex]],
       ['SOURCE_DATA', stats[:collection_uuid]],
       ['PIPELINE_UUID', pi.pipeline_uuid],
       ['PIPELINE_RUN_UUID', pi.uuid],
       ['PIPELINE_RUN_START', (stats[:runtime][:started_at] rescue nil)],
       ['PIPELINE_RUN_FINISH', (stats[:runtime][:finished_at] rescue nil)],
       ['N_READS_RAW',
        (n_raw = picardas[:total_reads].inject(0,:+) rescue nil)],
       ['N_READS_MAPPED',
        (n_mapped = picardas[:reads_aligned_in_pairs].inject(0,:+) rescue nil)],
       ['PERCENT_READS_MAPPED',
        (100.0 * n_mapped / n_raw rescue nil)],
       ['N_READS_ON_TARGET',
        (n_on_target = stats[:alignment][:total_reads] - stats[:alignment][:filtered_reads] rescue nil)],
       ['PERCENT_READS_ON_TARGET',
        (100.0 * n_on_target / n_raw rescue nil)],
       ['PERCENT_TARGET_COVERAGE_1X',
        (100.0 * stats[:coverage][1] / stats[:coverage][0] rescue nil)],
       ['PERCENT_TARGET_COVERAGE_10X',
        (100.0 * stats[:coverage][10] / stats[:coverage][0] rescue nil)],
       ['PERCENT_TARGET_COVERAGE_20X',
        (100.0 * stats[:coverage][20] / stats[:coverage][0] rescue nil)],
       ['PERCENT_TARGET_COVERAGE_50X',
        (100.0 * stats[:coverage][50] / stats[:coverage][0] rescue nil)],
       ['PERCENT_TARGET_COVERAGE_100X',
        (100.0 * stats[:coverage][100] / stats[:coverage][0] rescue nil)]]

    stats
  end
end
