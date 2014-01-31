function provenance_sizing_fixup(divId) {
    var a = document.getElementById(divId);
    //a.style.height = String(window.innerHeight - (a.getBoundingClientRect().top + window.scrollY) - 2) + "px";
    a.style.width = String(window.innerWidth - 25) + "px";
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
    }
