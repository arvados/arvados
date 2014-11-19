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
  window.jobGraphMaxima = {};
 */
function processLogLineForChart( logLine ) {
    try {
        var match = logLine.match(/(\S+) (\S+) (\S+) (\S+) stderr crunchstat: (\S+) (.*) -- interval (.*)/);
        if( match ) {
            // the timestamp comes first
            var timestamp = match[1].replace('_','T');
            // for the series use the first word after 'crunchstat:'
            var series = match[5];
            // and append the task number (the 4th term)
            series += '-' + match[4]
            if( $.inArray( series, jobGraphSeries) < 0 ) {
                jobGraphSeries.push(series);
                jobGraphMaxima[series] = null;
                window.recreate = true;
            }
            var intervalData = match[7].trim().split(' ');
            var dt = parseFloat(intervalData[0]);
            var dsum = 0.0;
            for(var i=2; i < intervalData.length; i += 2 ) {
                dsum += parseFloat(intervalData[i]);
            }
            var datum = dsum/dt;
            if( datum !== 0 && ( jobGraphMaxima[series] === null || jobGraphMaxima[series] < datum ) ) {
                if( isJobSeriesRescalable(series) ) {
                    // use old maximum to get a scale conversion
                    var scaleConversion = jobGraphMaxima[series]/datum;
                    // set new maximum and rescale the series
                    jobGraphMaxima[series] = datum;
                    rescaleJobGraphSeries( series, scaleConversion );
                }
                // and special calculation for cpus
                if( /^cpu-/.test(series) ) {
                    // divide the stat by the number of cpus
                    var cpuCountMatch = match[6].match(/(\d+) cpus/);
                    if( cpuCountMatch ) {
                        datum = datum / cpuCountMatch[1];
                    }
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
                    jobGraphData[i]['raw-'+series] = match[7];
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
                entry['raw-'+series] = match[7];
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
                // add a 10 minute old null data point to keep the chart honest if the oldest point is less than 9.5 minutes old
                if( jobGraphData.length > 0
                      && (Date.parse( jobGraphData[0]['t'] ).valueOf() + 9.5*60000 > Date.parse( jobGraphData[jobGraphData.length-1]['t'] ).valueOf()) ) {
                    var tenMinutesBefore = (new Date(Date.parse( jobGraphData[jobGraphData.length-1]['t'] ).valueOf() - 600*1000)).toISOString().replace('Z','');
                    jobGraphData.unshift( { 't': tenMinutesBefore } );
                }
            }
            window.redraw = true;
        }
    } catch( err ) {
        console.log( 'Ignoring error trying to process log line: ' + err);
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
            for( i = 0; i < options.ykeys.length; i++ ) {
                var series = options.ykeys[i];
                var datum = options.data[index][series];
                s += "<div class='morris-hover-point' style='color: ";
                s += options.lineColors[i];
                s += "'>";
                s += options.labels[i];
                s += ": ";
                if ( !(typeof datum === 'undefined') ) {
                    if( isJobSeriesRescalable( series ) ) {
                        datum *= jobGraphMaxima[series];
                    }
                    if( parseFloat(datum) !== 0 ) {
                        if( /^cpu-/.test(series) ){
                            datum = $.number(datum * 100, 1) + '%';
                        } else if( datum < 10 ) {
                            datum = $.number(datum, 2);
                        } else {
                            datum = $.number(datum);
                        }
                        datum += ' (' + options.data[index]['raw-'+series] + ')';
                    }
                    s += datum;
                } else {
                    s += '-';
                }
                s += "</div> ";
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
    return !/^cpu-/.test(series);
}

$(document).on('arv-log-event', '#log_graph_div', function(event, eventData) {
    if( eventData.properties.text ) {
        processLogLineForChart( eventData.properties.text );
    }
} );

$(document).on('ready', function(){
    window.recreate = false;
    window.redraw = false;
    setInterval( function() {
        if( recreate ) {
            window.recreate = false;
            window.redraw = false;
            // series have changed, draw entirely new graph
            $('#log_graph_div').html('');
            createJobGraph('log_graph_div');
        } else if( redraw ) {
            window.redraw = false;
            jobGraph.setData( jobGraphData );
        }
    }, 5000);
});
