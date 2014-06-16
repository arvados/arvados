function addToLogViewer(logViewer, lines, taskState) {
    var re = /((\d\d\d\d)-(\d\d)-(\d\d))_((\d\d):(\d\d):(\d\d)) ([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}) (\d+) (\d+)? (.*)/;
    for (var a in lines) {
        var v = lines[a].match(re);
        if (v != null) {

            var ts = new Date(Date.UTC(v[2], v[3], v[4], v[6], v[7], v[8]));

            v11 = v[11];
            if (typeof v[11] === 'undefined') {
                v11 = '&nbsp;';
            }

            var message = v[12];
            var type = "";
            if (v11 != '&nbsp;') {
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
                    if (/^success in (\d+)/) {
                        taskState[v11] = "success";
                    }
                    if (/^failure \([^)]+\) (\d+)/) {
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
