/*
 * This js establishes a websockets connection with the API Server.
 */

/* Subscribe to websockets event log.  Do nothing if already connected. */
function subscribeToEventLog () {
    // if websockets are not supported by browser, do not subscribe for events
    websocketsSupported = ('WebSocket' in window);
    if (websocketsSupported == false) {
        return;
    }

    // check if websocket connection is already stored on the window
    event_log_disp = $(window).data("arv-websocket");
    if (event_log_disp == null) {
        // need to create new websocket and event log dispatcher
        websocket_url = $('meta[name=arv-websocket-url]').attr("content");
        if (websocket_url == null)
            return;

        event_log_disp = new WebSocket(websocket_url);

        event_log_disp.onopen = onEventLogDispatcherOpen;
        event_log_disp.onmessage = onEventLogDispatcherMessage;

        // store websocket in window to allow reuse when multiple divs subscribe for events
        $(window).data("arv-websocket", event_log_disp);
    }
}

/* Send subscribe message to the websockets server.  Without any filters
   arguments, this subscribes to all events */
function onEventLogDispatcherOpen(event) {
    this.send('{"method":"subscribe"}');
}

/* Trigger event for all applicable elements waiting for this event */
function onEventLogDispatcherMessage(event) {
    parsedData = JSON.parse(event.data);
    object_uuid = parsedData.object_uuid;

    if (!object_uuid) {
        return;
    }

    // if there are any listeners for this object uuid or "all", trigger the event
    matches = ".arv-log-event-listener[data-object-uuid=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuids~=\"" + object_uuid + "\"],.arv-log-event-listener[data-object-uuid=\"all\"],.arv-log-event-listener[data-object-kind=\"" + parsedData.object_kind + "\"]";
    $(matches).trigger('arv-log-event', parsedData);
}

/* Automatically connect if there are any elements on the page that want to
   receive event log events. */
$(document).on('ajax:complete ready', function() {
    var a = $('.arv-log-event-listener');
    if (a.length > 0) {
        subscribeToEventLog();
    }
});

/* Assumes existence of:
  window.jobGraphData = [];
  window.jobGraphSeries = [];
  window.jobGraphSortedSeries = [];
  window.jobGraphMaxima = {};
 */
function processLogLineForChart( logLine ) {
    try {
        var match = logLine.match(/^(\S+) (\S+) (\S+) (\S+) stderr crunchstat: (\S+) (.*)/);
        if( !match ) {
            match = logLine.match(/^((?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{1,2} \d\d:\d\d:\d\d \d{4}) (\S+) (\S+) (\S+) stderr crunchstat: (\S+) (.*)/);
            if( match ) {
                match[1] = (new Date(match[1] + ' UTC')).toISOString().replace('Z','');
            }
        }
        if( match ) {
            var rawDetailData = '';
            var datum = null;

            // the timestamp comes first
            var timestamp = match[1].replace('_','T');

            // we are interested in "-- interval" recordings
            var intervalMatch = match[6].match(/(.*) -- interval (.*)/);
            if( intervalMatch ) {
                var intervalData = intervalMatch[2].trim().split(' ');
                var dt = parseFloat(intervalData[0]);
                var dsum = 0.0;
                for(var i=2; i < intervalData.length; i += 2 ) {
                    dsum += parseFloat(intervalData[i]);
                }
                datum = dsum/dt;

                if( datum < 0 ) {
                    // not interested in negative deltas
                    return;
                }

                rawDetailData = intervalMatch[2];

                // for the series name use the task number (4th term) and then the first word after 'crunchstat:'
                var series = 'T' + match[4] + '-' + match[5];

                // special calculation for cpus
                if( /-cpu$/.test(series) ) {
                    // divide the stat by the number of cpus
                    var cpuCountMatch = intervalMatch[1].match(/(\d+) cpus/);
                    if( cpuCountMatch ) {
                        datum = datum / cpuCountMatch[1];
                    }
                }

                addJobGraphDatum( timestamp, datum, series, rawDetailData );
            } else {
                // we are also interested in memory ("mem") recordings
                var memoryMatch = match[6].match(/(\d+) cache (\d+) swap (\d+) pgmajfault (\d+) rss/);
                if( memoryMatch ) {
                    rawDetailData = match[6];
                    // one datapoint for rss and one for swap - only show the rawDetailData for rss
                    addJobGraphDatum( timestamp, parseInt(memoryMatch[4]), 'T' + match[4] + "-rss", rawDetailData );
                    addJobGraphDatum( timestamp, parseInt(memoryMatch[2]), 'T' + match[4] + "-swap", '' );
                } else {
                    // not interested
                    return;
                }
            }

            window.redraw = true;
        }
    } catch( err ) {
        console.log( 'Ignoring error trying to process log line: ' + err);
    }
}

function addJobGraphDatum(timestamp, datum, series, rawDetailData) {
    // check for new series
    if( $.inArray( series, jobGraphSeries ) < 0 ) {
        var newIndex = jobGraphSeries.push(series) - 1;
        jobGraphSortedSeries.push(newIndex);
        jobGraphSortedSeries.sort( function(a,b) {
            var matchA = jobGraphSeries[a].match(/^T(\d+)-(.*)/);
            var matchB = jobGraphSeries[b].match(/^T(\d+)-(.*)/);
            var termA = ('000000' + matchA[1]).slice(-6) + matchA[2];
            var termB = ('000000' + matchB[1]).slice(-6) + matchB[2];
            return termA > termB ? 1 : -1;
        });
        jobGraphMaxima[series] = null;
        window.recreate = true;
    }

    if( datum !== 0 && ( jobGraphMaxima[series] === null || jobGraphMaxima[series] < datum ) ) {
        if( isJobSeriesRescalable(series) ) {
            // use old maximum to get a scale conversion
            var scaleConversion = jobGraphMaxima[series]/datum;
            // set new maximum and rescale the series
            jobGraphMaxima[series] = datum;
            rescaleJobGraphSeries( series, scaleConversion );
        }
    }

    // scale
    var scaledDatum = null;
    if( isJobSeriesRescalable(series) && jobGraphMaxima[series] !== null && jobGraphMaxima[series] !== 0 ) {
        scaledDatum = datum/jobGraphMaxima[series]
    } else {
        scaledDatum = datum;
    }
    // identify x axis point, searching from the end of the array (most recent)
    var found = false;
    for( var i = jobGraphData.length - 1; i >= 0; i-- ) {
        if( jobGraphData[i]['t'] === timestamp ) {
            found = true;
            jobGraphData[i][series] = scaledDatum;
            jobGraphData[i]['raw-'+series] = rawDetailData;
            break;
        } else if( jobGraphData[i]['t'] < timestamp  ) {
            // we've gone far enough back in time and this data is supposed to be sorted
            break;
        }
    }
    // index counter from previous loop will have gone one too far, so add one
    var insertAt = i+1;
    if(!found) {
        // create a new x point for this previously unrecorded timestamp
        var entry = { 't': timestamp };
        entry[series] = scaledDatum;
        entry['raw-'+series] = rawDetailData;
        jobGraphData.splice( insertAt, 0, entry );
        var shifted = [];
        // now let's see about "scrolling" the graph, dropping entries that are too old (>10 minutes)
        while( jobGraphData.length > 0
                 && (Date.parse( jobGraphData[0]['t'] ).valueOf() + 10*60000 < Date.parse( jobGraphData[jobGraphData.length-1]['t'] ).valueOf()) ) {
            shifted.push(jobGraphData.shift());
        }
        if( shifted.length > 0 ) {
            // from those that we dropped, are any of them maxima? if so we need to rescale
            jobGraphSeries.forEach( function(series) {
                // test that every shifted entry in this series was either not a number (in which case we don't care)
                // or else approximately (to 2 decimal places) smaller than the scaled maximum (i.e. 1),
                // because otherwise we just scrolled off something that was a maximum point
                // and so we need to recalculate a new maximum point by looking at all remaining displayed points in the series
                if( isJobSeriesRescalable(series) && jobGraphMaxima[series] !== null
                      && !shifted.every( function(e) { return( !$.isNumeric(e[series]) || e[series].toFixed(2) < 1.0 ) } ) ) {
                    // check the remaining displayed points and find the new (scaled) maximum
                    var seriesMax = null;
                    jobGraphData.forEach( function(entry) {
                        if( $.isNumeric(entry[series]) && (seriesMax === null || entry[series] > seriesMax)) {
                            seriesMax = entry[series];
                        }
                    });
                    if( seriesMax !== null && seriesMax !== 0 ) {
                        // set new actual maximum using the new maximum as the conversion conversion and rescale the series
                        jobGraphMaxima[series] *= seriesMax;
                        var scaleConversion = 1/seriesMax;
                        rescaleJobGraphSeries( series, scaleConversion );
                    }
                    else {
                        // we no longer have any data points displaying for this series
                        jobGraphMaxima[series] = null;
                    }
                }
            });
        }
        // add a 10 minute old null data point to keep the chart honest if the oldest point is less than 9.9 minutes old
        if( jobGraphData.length > 0
              && (Date.parse( jobGraphData[0]['t'] ).valueOf() + 9.9*60000 > Date.parse( jobGraphData[jobGraphData.length-1]['t'] ).valueOf()) ) {
            var tenMinutesBefore = (new Date(Date.parse( jobGraphData[jobGraphData.length-1]['t'] ).valueOf() - 600*1000)).toISOString().replace('Z','');
            jobGraphData.unshift( { 't': tenMinutesBefore } );
        }
    }

}

function createJobGraph(elementName) {
    delete jobGraph;
    var emptyGraph = false;
    if( jobGraphData.length === 0 ) {
        // If there is no data we still want to show an empty graph,
        // so add an empty datum and placeholder series to fool it into displaying itself.
        // Note that when finally a new series is added, the graph will be recreated anyway.
        jobGraphData.push( {} );
        jobGraphSeries.push( '' );
        emptyGraph = true;
    }
    var graphteristics = {
        element: elementName,
        data: jobGraphData,
        ymax: 1.0,
        yLabelFormat: function () { return ''; },
        xkey: 't',
        ykeys: jobGraphSeries,
        labels: jobGraphSeries,
        resize: true,
        hideHover: 'auto',
        parseTime: true,
        hoverCallback: function(index, options, content) {
            var s = "<div class='morris-hover-row-label'>";
            s += options.data[index][options.xkey];
            s += "</div> ";
            for( i = 0; i < jobGraphSortedSeries.length; i++ ) {
                var sortedIndex = jobGraphSortedSeries[i];
                var series = options.ykeys[sortedIndex];
                var datum = options.data[index][series];
                var point = ''
                point += "<div class='morris-hover-point' style='color: ";
                point += options.lineColors[sortedIndex];
                point += "'>";
                var labelMatch = options.labels[sortedIndex].match(/^T(\d+)-(.*)/);
                point += 'Task ' + labelMatch[1] + ' ' + labelMatch[2];
                point += ": ";
                if ( datum !== undefined ) {
                    if( isJobSeriesRescalable( series ) ) {
                        datum *= jobGraphMaxima[series];
                    }
                    if( parseFloat(datum) !== 0 ) {
                        if( /-cpu$/.test(series) ){
                            datum = $.number(datum * 100, 1) + '%';
                        } else if( datum < 10 ) {
                            datum = $.number(datum, 2);
                        } else {
                            datum = $.number(datum);
                        }
                        if(options.data[index]['raw-'+series]) {
                            datum += ' (' + options.data[index]['raw-'+series] + ')';
                        }
                    }
                    point += datum;
                } else {
                    continue;
                }
                point += "</div> ";
                s += point;
            }
            return s;
        }
    }
    if( emptyGraph ) {
        graphteristics['axes'] = false;
        graphteristics['parseTime'] = false;
        graphteristics['hideHover'] = 'always';
    }
    window.jobGraph = Morris.Line( graphteristics );
    if( emptyGraph ) {
        jobGraphData = [];
        jobGraphSeries = [];
    }
}

function rescaleJobGraphSeries( series, scaleConversion ) {
    if( isJobSeriesRescalable() ) {
        $.each( jobGraphData, function( i, entry ) {
            if( entry[series] !== null && entry[series] !== undefined ) {
                entry[series] *= scaleConversion;
            }
        });
    }
}

// that's right - we never do this for the 'cpu' series, which will always be between 0 and 1 anyway
function isJobSeriesRescalable( series ) {
    return !/-cpu$/.test(series);
}

$(document).on('arv-log-event', '#log_graph_div', function(event, eventData) {
    if( eventData.properties.text ) {
        processLogLineForChart( eventData.properties.text );
    }
} );

$(document).on('ready ajax:complete', function() {
    $('#log_graph_div').not('.graph-is-setup').addClass('graph-is-setup').each( function( index, graph_div ) {
        window.jobGraphData = [];
        window.jobGraphSeries = [];
        window.jobGraphSortedSeries = [];
        window.jobGraphMaxima = {};
        window.recreate = false;
        window.redraw = false;

        createJobGraph($(graph_div).attr('id'));

        $(document).trigger('ajax:send');
        $.get('/jobs/' + $(graph_div).data('object-uuid') + '/push_logs.js');

        setInterval( function() {
            if( recreate ) {
                window.recreate = false;
                window.redraw = false;
                // series have changed, draw entirely new graph
                $(graph_div).html('');
                createJobGraph($(graph_div).attr('id'));
            } else if( redraw ) {
                window.redraw = false;
                jobGraph.setData( jobGraphData );
            }
        }, 5000);
    });
});
