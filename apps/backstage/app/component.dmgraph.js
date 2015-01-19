module.exports = DataManagerGraph;

var _ = require('lodash');
var m = require('mithril');
// var Morris = require('morrisjs'); /* It's currently a global. */

function DataManagerGraph() {};
DataManagerGraph.controller = function(opts) {
    opts.connection.api('Log', 'list', {
        filters: [['event_type', '=', 'experimental-data-manager-report']],
        order: ['created_at DESC'],
    }).then(this.logs = m.prop([])).then(m.redraw);
    this.configGraph = function configGraph(element, isInitialized, context) {
        var seriesLabel = {};
        var data = _.compact(this.logs().map(function(item){
            try {
                var p = item().properties;
                var runId = '' + p.run_info.pid + '/' + p.run_info.start_time;
                var pt = {
                    seriesLabel: '' + p.run_info.pid,
                    collectionsRead: p.collection_info.collections_read,
                    logCreatedAt: toDate(item().created_at),
                    runStartTime: toDate(p.run_info.start_time),
                };
                pt[runId] = pt.collectionsRead;
                seriesLabel[runId] = p.run_info.pid;
                return pt;
            } catch(e) {
                return null;
            }
        }));
        var chartopts = {
            element: element,
            hoverCallback: morrisHoverCallback,
            xkey: 'logCreatedAt',
            ykeys: _.keys(seriesLabel),
            labels: _.values(seriesLabel),
            resize: true,
        };
        if (!isInitialized || !_.isEqual(chartopts, context.chartopts)) {
            context.chartopts = chartopts;
            context.chart = new Morris.Line(_.merge({data:data}, chartopts));
        } else if (!context.chart) {
            // Initialization crashed, no chart?
        } else if (!_.isEqual(data, context.data)) {
            context.chart.setData(data);
        }
        context.data = data;
    }.bind(this);
    function toDate(timestamp) {
        // 2015-01-13T01:13:52.281556508Z -> 2015-01-13 01:13:52.281
        return timestamp.match(/([-\d]{10})T([:\.\d]{8,12})/).slice(1).join(' ');
    }
    function morrisHoverCallback(index, options, content, row) {
        return ''+row.seriesLabel+': '+row.collectionsRead+' collections read @ '+row.logCreatedAt;
    }
};
DataManagerGraph.view = function(ctrl) {
    return ctrl.logs().length==0 ? [] : m('div', {
        config: ctrl.configGraph,
        style: { width: '100%', height: 200 },
    });
};
