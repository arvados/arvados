function provenance_sizing_fixup(divId, svgId) {
    var a = document.getElementById(divId);
    var b = document.getElementById("_" + divId + "_padding");
    var c = document.getElementById("_" + divId + "_center");
    var g = document.getElementById(svgId);
    
    var h = window.innerHeight - a.getBoundingClientRect().top - 2;
    var max = window.innerHeight - 100;
    var gh = parseFloat(g.getAttribute("height"));
    //var sh = (a.scrollTopMax > 0) ? a.scrollHeight : 0;
    //gh = (gh > sh) ? gh : sh;
    gh += 25;
    if (h < 0) { h = max; }
    var height;
    if (gh < h) {
        if (gh > max) { gh = max; }
        height = String(gh) + "px";
    }
    else {
        if (h > max) { h = max; }
        height = String(h) + "px";
    }
    a.style.height = height;
    b.style.paddingTop = height;

    w = window.innerWidth - 25;
    a.style.width = String(w) + "px";
    gw = parseFloat(g.getAttribute("width"));
    if (gw < w) {
        c.style.paddingLeft = String((w - gw)/2.0) + "px";
    }
    else {
        c.style.paddingLeft = "0px";
    }
}

function graph_zoom(divId, svgId, scale) {
    var pg = document.getElementById(divId);
    vcenter = (pg.scrollTop + (pg.scrollHeight - pg.scrollTopMax)/2.0) / pg.scrollHeight;
    hcenter = (pg.scrollLeft + (pg.scrollWidth - pg.scrollLeftMax)/2.0) / pg.scrollWidth;
    var g = document.getElementById(svgId);
    g.setAttribute("height", parseFloat(g.getAttribute("height")) * scale);
    g.setAttribute("width", parseFloat(g.getAttribute("width")) * scale);
    pg.scrollTop = (vcenter * pg.scrollHeight) - (pg.scrollHeight - pg.scrollTopMax)/2.0;
    pg.scrollLeft = (hcenter * pg.scrollWidth) - (pg.scrollWidth - pg.scrollLeftMax)/2.0;
    provenance_sizing_fixup(divId, svgId);
}
