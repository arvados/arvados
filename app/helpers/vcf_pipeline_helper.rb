module VcfPipelineHelper
  def reset_vcf_pipeline_invocation(pi, input_manifest)
    params = {
      'PICARD_ZIP' => '7a4073e29bfa87154b7102e75668c454+83+K@van',
      'GATK_BUNDLE' => '0a37aaf212464efa2a77ff9ba51c0148+10524+K@van',
      'GATK_TAR_BZ2' => '482ebab0408e173370c499f0b7c00878+93+K@van',
      'BWA' => '73be5598809c66f260fedd253c8608bd+67+K@van',
      'SAM' => '55d2115faa608eb95dab4f875b7511b1+72+K@van',
      'REGIONS' => 'e52c086f41c2f089d88ec2bbd45355d3+87+K@van/SeqCap_EZ_Exome_v2.hg19.bed',
      'STAND_CALL_CONF' => '4.0',
      'STAND_EMIT_CONF' => '4.0',
      "bwa/INPUT" => input_manifest
    }
    pi.components = Pipeline.find(pi.pipeline_uuid).components
    pi.update_job_parameters(params)
    pi.active = true
    pi.success = nil
  end
end
