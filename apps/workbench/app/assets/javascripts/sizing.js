function graph_zoom(divId, svgId, scale) {
    var pg = document.getElementById(divId);
    vcenter = (pg.scrollTop + (pg.scrollHeight - pg.scrollTopMax)/2.0) / pg.scrollHeight;
    hcenter = (pg.scrollLeft + (pg.scrollWidth - pg.scrollLeftMax)/2.0) / pg.scrollWidth;
    var g = document.getElementById(svgId);
    g.setAttribute("height", parseFloat(g.getAttribute("height")) * scale);
    g.setAttribute("width", parseFloat(g.getAttribute("width")) * scale);
    pg.scrollTop = (vcenter * pg.scrollHeight) - (pg.scrollHeight - pg.scrollTopMax)/2.0;
    pg.scrollLeft = (hcenter * pg.scrollWidth) - (pg.scrollWidth - pg.scrollLeftMax)/2.0;
    smart_scroll_fixup();
}

function smart_scroll_fixup(s) {

    if (s != null && s.type == 'shown.bs.tab') {
        s = [s.target];
    }
    else {
        s = $(".smart-scroll");
    }

    s.each(function(i, a) {
        a = $(a);
        var h = window.innerHeight - a.offset().top - a.attr("data-smart-scroll-padding-bottom");
        height = String(h) + "px";
        a.css('max-height', height);
    });
}

$(window).on('load ready resize scroll ajax:complete', smart_scroll_fixup);
$(document).on('shown.bs.tab', 'ul.nav-tabs > li > a', smart_scroll_fixup);
