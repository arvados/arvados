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
            var ret = ''
            if (s >= 86400) ret += Math.floor(s/86400) + 'd'
            if (s >= 3600) ret += Math.floor(s/3600)%24 + 'h'
            if (s >= 60) ret += Math.floor(s/60)%60 + 'm'
            ret += Math.floor(s)%60 + 's'
            // finally, strip trailing zeroes: 1d0m0s -> 1d
            return ret.replace(/(\D)(0\D)*$/, '$1')
        },
        date: function(s) {
	    return new Date(s).toLocaleString();
        },
    }
    var ticker = {
        time: function(min, max, pixels, opts, dg) {
            var max_ticks = Math.floor(pixels / opts('pixelsPerLabel'))
            var natural = [1, 5, 10, 30, 60,
                           120, 300, 600, 1800, 3600,
                           7200, 14400, 43200, 86400]
            var interval = natural.shift()
            while (max>min && (max-min)/interval > max_ticks) {
                interval = natural.shift() || (interval * 2)
            }
            var ticks = []
            for (var i=Math.ceil(min/interval)*interval; i<=max; i+=interval) {
                ticks.push({v: i, label: fmt.date(i)})
            }
            return ticks
        },
    }
    chartdata.forEach(function(section, section_idx) {
        var chartDiv = document.getElementById("chart");
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
            chartDiv.appendChild(div);
            chart.options.valueFormatter = function(y) {
            }
            chart.options.axes = {
                x: {
                    axisLabelFormatter: fmt.date,
                    valueFormatter: fmt.date,
                    ticker: ticker.time,
		    axisLabelWidth: 200,
		    pixelsPerLabel: 100
                },
                y: {
                    axisLabelFormatter: fmt.iso,
                    valueFormatter: fmt.iso,
                },
            }
            var div2 = document.createElement('div');
            div2.setAttribute('style', 'width: 150px; height: 150px');
            chart.options.labelsDiv = div2;
            chart.options.labelsSeparateLines = true;

            var div3 = document.createElement('div');
            div3.setAttribute('style', 'display: flex; padding-bottom: 16px');
            div3.appendChild(div);
            div3.appendChild(div2);
            chartDiv.appendChild(div3);

            charts[id] = new Dygraph(div, chart.data, chart.options);
        });
    });

    var sync = Dygraph.synchronize(Object.values(charts), {range: false});

    if (typeof window.debug === 'undefined')
        window.debug = {};
    window.debug.charts = charts;
};
