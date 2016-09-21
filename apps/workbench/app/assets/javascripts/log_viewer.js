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

function addToLogViewer(logViewer, lines) {
    var items = [];
    for (var a in lines) {
      items.push({
        message: lines[a].replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      });
    }
    logViewer.add(items);
}
