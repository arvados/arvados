cwlVersion: v1.0
class: ExpressionTool
requirements:
  InlineJavascriptRequirement: {}
inputs:
  dir: Directory
outputs:
  out: Directory[]
expression: |
  ${
    var samples = {};
    var pattern = /^(.+)(_S[0-9]{1,3}_)(.+)$/;
    for (var i = 0; i < inputs.dir.listing.length; i++) {
      var file = inputs.dir.listing[i];
      var groups = file.basename.match(pattern);
      if (groups) {
        var sampleid = groups[1];
        if (!samples[sampleid]) {
          samples[sampleid] = [];
        }
        samples[sampleid].push(file);
      }
    }
    var dirs = [];
    for (var sampleid in samples) {
      dirs.push({"class": "Directory",
                 "basename": sampleid,
                 "listing": samples[sampleid]});
    }
    return {"out": dirs};
  }