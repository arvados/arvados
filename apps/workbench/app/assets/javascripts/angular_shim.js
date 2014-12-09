// Compile any new HTML content that was loaded via jQuery.ajax().
// Currently this only works for tabs because they emit an
// arv:pane:loaded event after updating the DOM.

$(document).on('arv:pane:loaded', function(event, $updatedElement) {
    if ($updatedElement) {
        angular.element($updatedElement).injector().invoke(function($compile) {
            var scope = angular.element($updatedElement).scope();
            $compile($updatedElement)(scope);
        });
    }
});
