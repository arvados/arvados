module.exports = {
    entry: {
        app: './app',
    },
    output: {
        directory: 'dist',
        filename: 'dist/[name]'+(process.argv.indexOf('-p')>=0 ? '.min' : '')+'.js',
    },
};
