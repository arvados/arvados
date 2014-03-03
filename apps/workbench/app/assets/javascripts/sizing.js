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
    //console.log(s);
    if (s != null && s.type == 'shown.bs.tab') {
        s = [s.target];
    }
    else {
        s = $(".smart-scroll");
    }
    //console.log(s);
    for (var i = 0; i < s.length; i++) {
        a = s[i];
        var h = window.innerHeight - a.getBoundingClientRect().top - 20;
        height = String(h) + "px";
        a.style.height = height;
    }
}

$(window).on('load resize scroll', smart_scroll_fixup);
