function newTaskState() {
    return {"complete_count": 0,
            "failure_count": 0,
            "task_count": 0,
            "incomplete_count": 0,
            "nodes": []};
}

function addToLogViewer(logViewer, lines, taskState) {
    var re = /((\d\d\d\d)-(\d\d)-(\d\d))_((\d\d):(\d\d):(\d\d)) ([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}) (\d+) (\d+)? (.*)/;

    var items = [];
    var count = logViewer.items.length;
    for (var a in lines) {
        var v = lines[a].match(re);
        if (v != null) {

            var ts = new Date(Date.UTC(v[2], v[3]-1, v[4], v[6], v[7], v[8]));

            v11 = v[11];
            if (typeof v[11] === 'undefined') {
                v11 = "";
            } else {
                v11 = Number(v11);
            }

            var message = v[12];
            var type = "";
            var node = "";
            var slot = "";
            if (v11 !== "") {
                if (!taskState.hasOwnProperty(v11)) {
                    taskState[v11] = {};
                    taskState.task_count += 1;
                }

                if (/^stderr /.test(message)) {
                    message = message.substr(7);
                    if (/^crunchstat: /.test(message)) {
                        type = "crunchstat";
                        message = message.substr(12);
                    } else if (/^srun: /.test(message) || /^slurmd/.test(message)) {
                        type = "task-dispatch";
                    } else {
                        type = "task-print";
                    }
                } else {
                    var m;
                    if (m = /^success in (\d+) second/.exec(message)) {
                        taskState[v11].outcome = "success";
                        taskState[v11].runtime = Number(m[1]);
                        taskState.complete_count += 1;
                    }
                    else if (m = /^failure \(\#\d+, (temporary|permanent)\) after (\d+) second/.exec(message)) {
                        taskState[v11].outcome = "failure";
                        taskState[v11].runtime = Number(m[2]);
                        taskState.failure_count += 1;
                        if (m[1] == "permanent") {
                            taskState.incomplete_count += 1;
                        }
                    }
                    else if (m = /^child \d+ started on ([^.]*)\.(\d+)/.exec(message)) {
                        taskState[v11].node = m[1];
                        taskState[v11].slot = m[2];
                        if (taskState.nodes.indexOf(m[1], 0) == -1) {
                            taskState.nodes.push(m[1]);
                        }
                        for (var i in items) {
                            if (i > 0) {
                                if (items[i].taskid === v11) {
                                    items[i].node = m[1];
                                    items[i].slot = m[2];
                                }
                            }
                        }
                    }
                    type = "task-dispatch";
                }
                node = taskState[v11].node;
                slot = taskState[v11].slot;
            } else {
                type = "crunch";
            }

            items.push({
                id: count,
                ts: ts,
                timestamp: ts.toLocaleDateString() + " " + ts.toLocaleTimeString(),
                taskid: v11,
                node: node,
                slot: slot,
                message: message.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'),
                type: type
            });
            count += 1;
        } else {
            console.log("Did not parse line " + a + ": " + lines[a]);
        }
    }
    logViewer.add(items);
}

function sortById(a, b, opt) {
    a = a.values();
    b = b.values();

    if (a["id"] > b["id"]) {
        return 1;
    }
    if (a["id"] < b["id"]) {
        return -1;
    }
    return 0;
}

function sortByTask(a, b, opt) {
    var aa = a.values();
    var bb = b.values();

    if (aa["taskid"] === "" && bb["taskid"] !== "") {
        return -1;
    }
    if (aa["taskid"] !== "" && bb["taskid"] === "") {
        return 1;
    }

    if (aa["taskid"] !== "" && bb["taskid"] !== "") {
        if (aa["taskid"] > bb["taskid"]) {
            return 1;
        }
        if (aa["taskid"] < bb["taskid"]) {
            return -1;
        }
    }

    return sortById(a, b, opt);
}

function sortByNode(a, b, opt) {
    var aa = a.values();
    var bb = b.values();

    if (aa["node"] === "" && bb["node"] !== "") {
        return -1;
    }
    if (aa["node"] !== "" && bb["node"] === "") {
        return 1;
    }

    if (aa["node"] !== "" && bb["node"] !== "") {
        if (aa["node"] > bb["node"]) {
            return 1;
        }
        if (aa["node"] < bb["node"]) {
            return -1;
        }
    }

    if (aa["slot"] !== "" && bb["slot"] !== "") {
        if (aa["slot"] > bb["slot"]) {
            return 1;
        }
        if (aa["slot"] < bb["slot"]) {
            return -1;
        }
    }

    return sortById(a, b, opt);
}


function dumbPluralize(n, s, p) {
    if (typeof p === 'undefined') {
        p = "s";
    }
    if (n == 0 || n > 1) {
        return n + " " + (s + p);
    } else {
        return n + " " + s;
    }
}

function generateJobOverview(id, logViewer, taskState) {
    var html = "";

    if (logViewer.items.length > 2) {
        var first = logViewer.items[1];
        var last = logViewer.items[logViewer.items.length-1];
        var duration = (last.values().ts.getTime() - first.values().ts.getTime()) / 1000;

        var hours = 0;
        var minutes = 0;
        var seconds;

        if (duration >= 3600) {
            hours = Math.floor(duration / 3600);
            duration -= (hours * 3600);
        }
        if (duration >= 60) {
            minutes = Math.floor(duration / 60);
            duration -= (minutes * 60);
        }
        seconds = duration;

        var tcount = taskState.task_count;

        html += "<p>";
        html += "Started at " + first.values().timestamp + ".  ";
        html += "Ran " + dumbPluralize(tcount, " task") + " over ";
        if (hours > 0) {
            html += dumbPluralize(hours, " hour");
        }
        if (minutes > 0) {
            html += " " + dumbPluralize(minutes, " minute");
        }
        if (seconds > 0) {
            html += " " + dumbPluralize(seconds, " second");
        }

        html += " using " + dumbPluralize(taskState.nodes.length, " node");

        html += ".  " + dumbPluralize(taskState.complete_count, "task") + " completed";
        html += ",  " + dumbPluralize(taskState.incomplete_count, "task") +  " incomplete";
        html += " (" + dumbPluralize(taskState.failure_count, " failure") + ")";

        html += ".  Finished at " + last.values().timestamp + ".";
        html += "</p>";
    } else {
       html = "<p>Job log is empty or failed to load.</p>";
    }

    $(id).html(html);
}

function gotoPage(n, logViewer, page, id) {
    if (n < 0) { return; }
    if (n*page > logViewer.matchingItems.length) { return; }
    logViewer.page_offset = n;
    logViewer.show(n*page, page);
}

function updatePaging(id, logViewer, page) {
    var p = "";
    var i = logViewer.matchingItems.length;
    var n;
    for (n = 0; (n*page) < i; n += 1) {
        if (n == logViewer.page_offset) {
            p += "<span class='log-viewer-page-num'>" + (n+1) + "</span> ";
        } else {
            p += "<a href=\"#\" class='log-viewer-page-num log-viewer-page-" + n + "'>" + (n+1) + "</a> ";
        }
    }
    $(id).html(p);
    for (n = 0; (n*page) < i; n += 1) {
        (function(n) {
            $(".log-viewer-page-" + n).on("click", function() {
                gotoPage(n, logViewer, page, id);
                return false;
            });
        })(n);
    }

    if (logViewer.page_offset == 0) {
        $(".log-viewer-page-up").addClass("text-muted");
    } else {
        $(".log-viewer-page-up").removeClass("text-muted");
    }

    if (logViewer.page_offset == (n-1)) {
        $(".log-viewer-page-down").addClass("text-muted");
    } else {
        $(".log-viewer-page-down").removeClass("text-muted");
    }
}

function nextPage(logViewer, page, id) {
    gotoPage(logViewer.page_offset+1, logViewer, page, id);
}

function prevPage(logViewer, page, id) {
    gotoPage(logViewer.page_offset-1, logViewer, page, id);
}
