window.onload = function() {
    var charts = {};
    sections.forEach(function(section, section_idx) {
        var h1 = document.createElement('h1');
        h1.appendChild(document.createTextNode(section.label));
        document.body.appendChild(h1);
        section.charts.forEach(function(data, chart_idx) {
            // Skip chart if every series has zero data points
            if (0 == data.data.reduce(function(len, series) {
                return len + series.dataPoints.length;
            }, 0)) {
                return;
            }
            var id = 'chart-'+section_idx+'-'+chart_idx;
            var div = document.createElement('div');
            div.setAttribute('id', id);
            div.setAttribute('style', 'width: 100%; height: 150px');
            document.body.appendChild(div);
            charts[id] = new CanvasJS.Chart(id, data);
            charts[id].render();
        });
    });

    if (typeof window.debug === 'undefined')
        window.debug = {};
    window.debug.charts = charts;
};
