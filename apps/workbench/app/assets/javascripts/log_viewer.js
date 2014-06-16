function addToLogViewer(logViewer, lines, taskState) {
    var re = /((\d\d\d\d)-(\d\d)-(\d\d))_((\d\d):(\d\d):(\d\d)) ([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}) (\d+) (\d+)? (.*)/;
    for (var a in lines) {
        var v = lines[a].match(re);
        if (v != null) {

            var ts = new Date(Date.UTC(v[2], v[3], v[4], v[6], v[7], v[8]));

            v11 = v[11];
            if (typeof v[11] === 'undefined') {
                v11 = "";
            } else {
                v11 = Number(v11);
            }

            var message = v[12];
            var type = "";
            var node = "";
            if (v11 !== "") {
                if (!taskState.hasOwnProperty(v11)) {
                    taskState[v11] = {};
                }

                if (/^stderr /.test(message)) {
                    message = message.substr(7);
                    if (/^crunchstat: /.test(message)) {
                        type = "crunchstat";
                        message = message.substr(12);
                    } else if (/^srun: /.test(message) || /^slurmd/.test(message)) {
                        type = "task-dispatch";
                    } else {
                        type = "task-output";
                    }
                } else {
                    if (/^success /.test(message)) {
                        taskState[v11].outcome = "success";
                        taskState.success_count += 1;
                    }
                    else if (/^failure /.test(message)) {
                        taskState[v11].outcome = "failure";
                        taskState.failure_count += 1;
                    }
                    else {
                        var child = message.match(/^child \d+ started on (.*)/);
                        if (child != null) {
                            taskState[v11].node = child[1];
                            for (var i in logViewer.items) {
                                if (i > 0) {
                                    var val = logViewer.items[i].values();
                                    if (val.taskid === v11) {
                                        val.node = child[1];
                                        logViewer.items[i].values(val);
                                    }
                                }
                            }
                        }
                    }
                    type = "task-dispatch";
                }
                node = taskState[v11].node;
            } else {
                if (/^status: /.test(message)) {
                    type = "job-status";
                    message = message.substr(8);
                } else {
                    type = "crunch";
                }
            }

            logViewer.add({
                id: logViewer.items.length,
                ts: ts,
                timestamp: ts.toLocaleDateString() + " " + ts.toLocaleTimeString(),
                taskid: v11,
                node: node,
                message: message,
                type: type
            });

        } else {
            console.log("Did not parse: " + lines[a]);
        }
    }
    logViewer.update();
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

    return sortById(a, b, opt);
}


function dumbPluralize(n, s, p) {
    if (typeof p === 'undefined') {
        p = "s";
    }
    if (n == 0 || n > 1) {
        return (s + p);
    } else {
        return s;
    }
}

function generateJobOverview(id, logViewer, taskState) {
    var html = "";

    var first = logViewer.items[1];
    var last = logViewer.items[logViewer.items.length-1];

    {
        html += "<div>";
        html += "Started at " + first.values().timestamp;

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

        var tcount = taskState.success_count + taskState.failure_count;

        html += ".  " + tcount + dumbPluralize(tcount, " task") + " completed in ";
        if (hours > 0) {
            html += hours + dumbPluralize(hours, " hour");
        }
        if (minutes > 0) {
            html += " " + minutes + dumbPluralize(minutes, " minute");
        }
        if (seconds > 0) {
            html += " " + seconds + dumbPluralize(seconds, " second");
        }
        html += ".  " + taskState.success_count + dumbPluralize(taskState.success_count, " success", "es");
        html += ", " + taskState.failure_count + dumbPluralize(taskState.failure_count, " failure");

        html += ".  Completed at " + last.values().timestamp;
        html += "</div>";
    }

    $(id).html(html);
}