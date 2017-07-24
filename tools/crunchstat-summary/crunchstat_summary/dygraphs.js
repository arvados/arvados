// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

window.onload = function() {
    var charts = {};
    var fmt = {
        iso: function(y) {
            var s='';
            if (y > 1000000000) { y=y/1000000000; s='G'; }
            else if (y > 1000000) { y=y/1000000; s='M'; }
            else if (y > 1000) { y=y/1000; s='K'; }
            return y.toFixed(2).replace(/\.0+$/, '')+s;
        },
        time: function(s) {
            var u='s';
            if (s < 60) return s;
            var u = 'm'+(s%60)+'s';
            var m = Math.floor(s/60);
            if (m < 60) return ''+m+u;
            u = 'h'+(m%60)+u;
            var h = Math.floor(m/60);
            if (h < 24) return ''+h+u;
            u = 'd'+(h%24)+s;
            return ''+Math.floor(h/24)+u;
        },
    }
    chartdata.forEach(function(section, section_idx) {
        var h1 = document.createElement('h1');
        h1.appendChild(document.createTextNode(section.label));
        document.body.appendChild(h1);
        section.charts.forEach(function(chart, chart_idx) {
            // Skip chart if every series has zero data points
            if (0 == chart.data.reduce(function(len, series) {
                return len + series.length;
            }, 0)) {
                return;
            }
            var id = 'chart-'+section_idx+'-'+chart_idx;
            var div = document.createElement('div');
            div.setAttribute('id', id);
            div.setAttribute('style', 'width: 100%; height: 150px');
            document.body.appendChild(div);
            chart.options.valueFormatter = function(y) {
            }
            chart.options.axes = {
                x: {
                    axisLabelFormatter: fmt.time,
                    valueFormatter: fmt.time,
                },
                y: {
                    axisLabelFormatter: fmt.iso,
                    valueFormatter: fmt.iso,
                },
            }
            charts[id] = new Dygraph(div, chart.data, chart.options);
        });
    });

    if (typeof window.debug === 'undefined')
        window.debug = {};
    window.debug.charts = charts;
};
