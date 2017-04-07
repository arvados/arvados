$(document).ready(function(event) {
  $(".btn-collection-file-control").addClass("disabled");
  $(".tab-pane-Upload").addClass("disabled");
  $("#Upload-tab").attr("data-toggle", "disabled");
}).on('click', '.lock-collection-btn', function(event) {
  classes = $(event.target).attr('class')
  if(classes.includes("fa-lock")){
    $(".lock-collection-btn").removeClass("fa-lock");
    $(".lock-collection-btn").addClass("fa-unlock");
    $(".btn-collection-file-control").removeClass("disabled");
    $(".tab-pane-Upload").removeClass("disabled");
    $("#Upload-tab").attr("data-toggle", "tab");
  } else {
    $(".lock-collection-btn").removeClass("fa-unlock");
    $(".lock-collection-btn").addClass("fa-lock");
    $(".btn-collection-file-control").addClass("disabled");
    $(".tab-pane-Upload").addClass("disabled");
    $("#Upload-tab").attr("data-toggle", "disabled");
  }
});
