// -*- mode: javascript; js-indent-level: 4; indent-tabs-mode: nil; -*-
// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

var loaded_nodes_js;
$(function(){
    if (loaded_nodes_js) return; loaded_nodes_js = true;

    $('[data-showhide-selector]').on('click', function(e){
        var x = $($(this).attr('data-showhide-selector'));
        if (x.css('display') == 'none')
            x.show();
        else
            x.hide();
    });
    $('[data-showhide-default]').hide();
});
