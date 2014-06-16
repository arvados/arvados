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
            if (v11 !== "") {
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
                        taskState[v11] = "success";
                    }
                    else if (/^failure /.test(message)) {
                        taskState[v11] = "failure";
                    }
                    type = "task-dispatch";
                }
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
                timestamp: ts.toLocaleDateString() + " " + ts.toLocaleTimeString(),
                taskid: v11,
                message: message,
                type: type
            });

        } else {
            console.log("Did not parse: " + lines[a]);
        }
    }
    logViewer.update();
}

function sortByTaskThenId(a, b, opt) {
    a = a.values();
    b = b.values();

    if (a["taskid"] === "" && b["taskid"] !== "") {
        return -1;
    }
    if (a["taskid"] !== "" && b["taskid"] === "") {
        return 1;
    }

    if (a["taskid"] !== "" && b["taskid"] !== "") {
        if (a["taskid"] > b["taskid"]) {
            return 1;
        }
        if (a["taskid"] < b["taskid"]) {
            return -1;
        }
    }

    if (a["id"] > b["id"]) {
        return 1;
    }
    if (a["id"] < b["id"]) {
        return -1;
    }
    return 0;
}
