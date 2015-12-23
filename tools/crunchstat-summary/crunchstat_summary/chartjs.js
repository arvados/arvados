window.onload = function() {
    var options = {};
    chartData.forEach(function(data, idx) {
        var div = document.createElement('div');
        div.setAttribute('id', 'chart-'+idx);
        div.setAttribute('style', 'width: 100%; height: 150px');
        document.body.appendChild(div);
        var chart = new CanvasJS.Chart('chart-'+idx, data);
        chart.render();
    });
}
