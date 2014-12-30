define(['webdriverjs'], function(webdriverjs) {
    var client = webdriverjs.remote({
        desiredCapabilities: {
            // http://code.google.com/p/selenium/wiki/DesiredCapabilities
            browserName: 'phantomjs'
        },
        // webdriverjs has a lot of output which is generally useless
        // However, if anything goes wrong, remove this to see more details
        // logLevel: 'silent'
    });
    client.init();
    return client;
});
